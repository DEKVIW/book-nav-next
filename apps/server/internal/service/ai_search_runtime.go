package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// searchIntent is a lightweight parse of LLM intent JSON (legacy-compatible shape).
type searchIntent struct {
	Intent        string   `json:"intent"`
	Keywords      []string `json:"keywords"`
	RelatedTerms  []string `json:"related_terms"`
	CategoryHints []string `json:"category_hints"`
	SearchType    string   `json:"search_type"`
}

// shouldRunIntent: short exact queries skip LLM (faster, cheaper).
func shouldRunIntent(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	// rune length > 5 or contains spaces or Chinese question patterns
	n := 0
	for range q {
		n++
	}
	if n > 5 {
		return true
	}
	if strings.Contains(q, " ") || strings.Contains(q, "\t") {
		return true
	}
	for _, w := range []string{"怎么", "如何", "哪里", "为什么", "什么", "哪个", "推荐", "有没有"} {
		if strings.Contains(q, w) {
			return true
		}
	}
	// mixed alnum long token
	letters := 0
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letters++
		}
	}
	return letters >= 8
}

func chatCompletion(ctx context.Context, baseURL, apiKey, model, userMsg string, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = 256
	}
	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": userMsg},
		},
		"max_tokens":  maxTokens,
		"temperature": 0.2,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %s: %s", resp.Status, truncate(string(raw), 200))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("解析响应失败")
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("模型返回空内容")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

var jsonBlockRe = regexp.MustCompile(`(?s)\{.*\}`)

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	// strip markdown fences
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") {
		return s
	}
	m := jsonBlockRe.FindString(s)
	return m
}

func parseSearchIntent(raw string) (*searchIntent, error) {
	js := extractJSONObject(raw)
	if js == "" {
		return nil, fmt.Errorf("无 JSON")
	}
	var in searchIntent
	if err := json.Unmarshal([]byte(js), &in); err != nil {
		return nil, err
	}
	return &in, nil
}

func analyzeSearchIntent(ctx context.Context, cand *taskCandidate, query string) (*searchIntent, error) {
	if cand == nil || cand.Model == "" {
		return nil, fmt.Errorf("no candidate")
	}
	prompt := fmt.Sprintf(`你是网站导航搜索助手。分析用户意图，只返回 JSON，不要其它文字。

用户查询：%s

返回格式：
{"intent":"简短意图","keywords":["关键词1","关键词2"],"related_terms":["相关词"],"category_hints":["分类提示"],"search_type":"exact|fuzzy|semantic"}

要求：keywords 最多 5 个，related_terms 最多 3 个，用中文或用户语言，利于站名/描述匹配。`, query)
	raw, err := chatCompletion(ctx, cand.APIBaseURL, cand.APIKey, cand.Model, prompt, 200)
	if err != nil {
		return nil, err
	}
	return parseSearchIntent(raw)
}

// rerankWithLLM ranks candidate ids; returns validated order + optional summary.
// Only IDs present in items are kept; empty/invalid model output is an error (caller keeps fusion order).
func rerankWithLLM(ctx context.Context, cand *taskCandidate, query, intentSummary string, items []rerankItem, maxOut int) ([]int64, string, error) {
	if cand == nil || cand.Model == "" || len(items) == 0 {
		return nil, "", fmt.Errorf("no items")
	}
	if maxOut <= 0 {
		maxOut = 20
	}
	if len(items) > 40 {
		items = items[:40]
	}
	var b strings.Builder
	for i, it := range items {
		fmt.Fprintf(&b, "%d. id=%d | %s | %s\n", i+1, it.ID, truncate(it.Title, 40), truncate(it.Desc, 60))
	}
	intentLine := strings.TrimSpace(intentSummary)
	if intentLine == "" {
		intentLine = "(none)"
	}
	prompt := fmt.Sprintf(`你是网站导航排序助手。按与用户查询的相关性从高到低排序，只返回 JSON。

用户查询：%s
意图摘要：%s
候选（最多选 %d 个）：
%s
返回格式：
{"order":[id1,id2,id3],"summary":"一句话中文摘要"}

硬性规则：
1. order 只能使用上面出现的 id，禁止编造
2. 按相关度降序，最多 %d 个
3. 不要把明显无关的站排到最前`, query, intentLine, maxOut, b.String(), maxOut)

	raw, err := chatCompletion(ctx, cand.APIBaseURL, cand.APIKey, cand.Model, prompt, 400)
	if err != nil {
		return nil, "", err
	}
	js := extractJSONObject(raw)
	if js == "" {
		return nil, "", fmt.Errorf("no JSON")
	}
	var out struct {
		Order   []int64 `json:"order"`
		Summary string  `json:"summary"`
	}
	if err := json.Unmarshal([]byte(js), &out); err != nil {
		var out2 struct {
			Order   []any  `json:"order"`
			Summary string `json:"summary"`
		}
		if err2 := json.Unmarshal([]byte(js), &out2); err2 != nil {
			return nil, "", err
		}
		out.Summary = out2.Summary
		for _, v := range out2.Order {
			switch t := v.(type) {
			case float64:
				out.Order = append(out.Order, int64(t))
			case json.Number:
				i, _ := t.Int64()
				out.Order = append(out.Order, i)
			}
		}
	}
	if len(out.Order) == 0 {
		return nil, "", fmt.Errorf("empty order")
	}
	known := map[int64]bool{}
	for _, it := range items {
		known[it.ID] = true
	}
	var ordered []int64
	seen := map[int64]bool{}
	for _, id := range out.Order {
		if !known[id] || seen[id] {
			continue
		}
		seen[id] = true
		ordered = append(ordered, id)
		if len(ordered) >= maxOut {
			break
		}
	}
	if len(ordered) == 0 {
		return nil, "", fmt.Errorf("no valid ids")
	}
	return ordered, strings.TrimSpace(out.Summary), nil
}

type rerankItem struct {
	ID    int64
	Title string
	Desc  string
	URL   string
}
