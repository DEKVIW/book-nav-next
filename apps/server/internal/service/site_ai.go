package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
)

// TranslateText uses the AI "translate" task to turn text into Simplified Chinese.
func (s *JobService) TranslateText(ctx context.Context, text, field string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, apperr.New(apperr.Validation, "没有可翻译的内容")
	}
	if len([]rune(text)) > 2000 {
		text = string([]rune(text)[:2000])
	}
	aiOn, _ := s.settings.AIEnabled(ctx)
	if !aiOn {
		return nil, apperr.New(apperr.Validation, "请先在后台启用 AI")
	}
	providers, _ := s.settings.loadProviders(ctx)
	bindings := s.settings.loadTaskBindings(ctx)
	cand := s.settings.resolveTaskCandidate(providers, bindings, "translate")
	if cand == nil || cand.Model == "" {
		return nil, apperr.New(apperr.Validation, "未配置翻译模型，请到站点设置 → AI 配置")
	}
	label := "文本"
	switch field {
	case "title":
		label = "网站标题"
	case "description":
		label = "网站描述"
	}
	prompt := fmt.Sprintf(`你是网站导航助手。将下列%s译为简体中文。
要求：只输出译文，不要引号、不要解释；保留专有名词/品牌名合理写法；若已是通顺中文可轻微润色仍只输出结果。

原文：
%s`, label, text)
	out, err := chatCompletion(ctx, cand.APIBaseURL, cand.APIKey, cand.Model, prompt, 400)
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "翻译失败", err)
	}
	out = strings.TrimSpace(out)
	out = strings.Trim(out, "\"'")
	if out == "" {
		return nil, apperr.New(apperr.Internal, "模型返回空译文")
	}
	return map[string]any{
		"text":          out,
		"original":      text,
		"field":         field,
		"provider_name": cand.Name,
		"model":         cand.Model,
	}, nil
}

// EnhanceSiteInfo uses the AI "site_info" task to fill title/description for quick-add.
func (s *JobService) EnhanceSiteInfo(ctx context.Context, pageURL, title, description string) (map[string]any, error) {
	pageURL = strings.TrimSpace(pageURL)
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if pageURL == "" && title == "" {
		return nil, apperr.New(apperr.Validation, "请先填写网址或标题")
	}
	aiOn, _ := s.settings.AIEnabled(ctx)
	if !aiOn {
		return nil, apperr.New(apperr.Validation, "请先在后台启用 AI")
	}
	providers, _ := s.settings.loadProviders(ctx)
	bindings := s.settings.loadTaskBindings(ctx)
	cand := s.settings.resolveTaskCandidate(providers, bindings, "site_info")
	if cand == nil || cand.Model == "" {
		return nil, apperr.New(apperr.Validation, "未配置网站信息补全模型，请到站点设置 → AI 配置")
	}
	prompt := fmt.Sprintf(`你是网站导航助手。根据已知信息补全适合导航卡片展示的中文标题与描述。
只返回 JSON，不要其它文字：
{"title":"简洁中文标题","description":"一句话中文描述，不超过80字"}

已知：
- URL: %s
- 现有标题: %s
- 现有描述: %s

要求：描述真实可用，不要编造不存在的功能；若信息不足写稳妥概括；标题勿过长。`, pageURL, title, description)
	raw, err := chatCompletion(ctx, cand.APIBaseURL, cand.APIKey, cand.Model, prompt, 300)
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "AI 补全失败", err)
	}
	js := extractJSONObject(raw)
	if js == "" {
		return nil, apperr.New(apperr.Internal, "模型未返回有效 JSON")
	}
	var parsed struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		return nil, apperr.New(apperr.Internal, "解析模型结果失败")
	}
	parsed.Title = strings.TrimSpace(parsed.Title)
	parsed.Description = strings.TrimSpace(parsed.Description)
	if parsed.Title == "" && parsed.Description == "" {
		return nil, apperr.New(apperr.Internal, "模型未给出标题或描述")
	}
	return map[string]any{
		"title":         parsed.Title,
		"description":   parsed.Description,
		"provider_name": cand.Name,
		"model":         cand.Model,
	}, nil
}
