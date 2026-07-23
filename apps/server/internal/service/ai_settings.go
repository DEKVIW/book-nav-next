package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
)

// AIProvider is a multi-line OpenAI-compatible endpoint.
type AIProvider struct {
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	APIBaseURL         string            `json:"api_base_url"`
	APIKey             string            `json:"api_key,omitempty"`
	APIKeyConfigured   bool              `json:"api_key_configured,omitempty"`
	InterfaceMode      string            `json:"interface_mode"`
	Enabled            bool              `json:"enabled"`
	Priority           int               `json:"priority"`
	ModelCatalog       []AIModelInfo     `json:"model_catalog,omitempty"`
	RecommendedModels  map[string]string `json:"recommended_models,omitempty"`
	ProbeLastAt        string            `json:"probe_last_at,omitempty"`
	ProbeError         string            `json:"probe_error,omitempty"`
	ProbeSignature     string            `json:"probe_signature,omitempty"`
}

type AIModelInfo struct {
	ID          string `json:"id"`
	Compatible  string `json:"compatible,omitempty"` // full | partial | unknown
	Description string `json:"description,omitempty"`
}

// AITaskBinding maps a task to auto or a specific provider/model.
type AITaskBinding struct {
	Mode       string `json:"mode"` // auto | manual
	ProviderID *int64 `json:"provider_id"`
	ModelName  string `json:"model_name"`
}

type AITaskTestResult struct {
	Status       string `json:"status"` // idle | success | error
	Message      string `json:"message"`
	ProviderID   *int64 `json:"provider_id,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	TestedAt     string `json:"tested_at,omitempty"`
	Protocol     string `json:"protocol,omitempty"`
}

var AITaskKeys = []string{"intent", "rerank", "translate", "site_info"}

var AITaskLabels = map[string]string{
	"intent":    "搜索意图分析",
	"rerank":    "搜索结果重排",
	"translate": "翻译",
	"site_info": "网站信息补全",
}

func defaultTaskBindings() map[string]AITaskBinding {
	out := map[string]AITaskBinding{}
	for _, k := range AITaskKeys {
		out[k] = AITaskBinding{Mode: "auto", ModelName: ""}
	}
	return out
}

func defaultTaskTestResults() map[string]AITaskTestResult {
	out := map[string]AITaskTestResult{}
	for _, k := range AITaskKeys {
		out[k] = AITaskTestResult{Status: "idle"}
	}
	return out
}

// AIState is the admin management aggregate.
type AIState struct {
	Providers     []AIProvider                  `json:"providers"`
	TaskBindings  map[string]AITaskBinding      `json:"task_bindings"`
	TaskTests     map[string]AITaskTestResult   `json:"task_test_results"`
	Effective     map[string]map[string]any     `json:"effective_tasks"`
	Summary       map[string]int                `json:"summary"`
	Enabled       bool                          `json:"enabled"`
	AllowAnon     bool                          `json:"allow_anonymous"`
	Temperature   float64                       `json:"temperature"`
	MaxTokens     int                           `json:"max_tokens"`
}

func (s *SettingsService) loadProviders(ctx context.Context) ([]AIProvider, error) {
	raw, err := s.repo.Get(ctx, "ai", "providers")
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []AIProvider{}, nil
	}
	var list []AIProvider
	if err := json.Unmarshal(raw, &list); err != nil {
		return []AIProvider{}, nil
	}
	return list, nil
}

func (s *SettingsService) saveProviders(ctx context.Context, list []AIProvider) error {
	return s.repo.Set(ctx, "ai", "providers", list)
}

func (s *SettingsService) loadTaskBindings(ctx context.Context) map[string]AITaskBinding {
	raw, _ := s.repo.Get(ctx, "ai", "task_bindings")
	out := defaultTaskBindings()
	if len(raw) == 0 {
		return out
	}
	var m map[string]AITaskBinding
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for _, k := range AITaskKeys {
		if b, ok := m[k]; ok {
			if b.Mode != "manual" {
				b.Mode = "auto"
			}
			out[k] = b
		}
	}
	return out
}

func (s *SettingsService) loadTaskTests(ctx context.Context) map[string]AITaskTestResult {
	raw, _ := s.repo.Get(ctx, "ai", "task_test_results")
	out := defaultTaskTestResults()
	if len(raw) == 0 {
		return out
	}
	var m map[string]AITaskTestResult
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for _, k := range AITaskKeys {
		if t, ok := m[k]; ok {
			out[k] = t
		}
	}
	return out
}

func maskProvider(p AIProvider) AIProvider {
	configured := strings.TrimSpace(p.APIKey) != ""
	p.APIKeyConfigured = configured
	if configured {
		p.APIKey = secretSentinel
	} else {
		p.APIKey = ""
	}
	return p
}

func (s *SettingsService) GetAIState(ctx context.Context) (*AIState, error) {
	ai, _ := s.repo.GetNamespace(ctx, "ai")
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	// migrate legacy single-line fields into a provider if empty
	if len(providers) == 0 {
		base := decodeStr(ai["api_base_url"], "")
		key := decodeStr(ai["api_key"], "")
		if base != "" || key != "" {
			providers = []AIProvider{{
				ID:            1,
				Name:          "默认提供方",
				APIBaseURL:    base,
				APIKey:        key,
				InterfaceMode: decodeStr(ai["interface_mode"], "auto"),
				Enabled:       true,
				Priority:      100,
			}}
			_ = s.saveProviders(ctx, providers)
		}
	}

	bindings := s.loadTaskBindings(ctx)
	tests := s.loadTaskTests(ctx)
	masked := make([]AIProvider, 0, len(providers))
	enabledCount := 0
	detected := 0
	for _, p := range providers {
		if p.Enabled {
			enabledCount++
		}
		if len(p.ModelCatalog) > 0 {
			detected++
		}
		masked = append(masked, maskProvider(p))
	}
	sort.SliceStable(masked, func(i, j int) bool {
		if masked[i].Priority == masked[j].Priority {
			return masked[i].ID < masked[j].ID
		}
		return masked[i].Priority < masked[j].Priority
	})

	effective := map[string]map[string]any{}
	for _, task := range AITaskKeys {
		cand := s.resolveTaskCandidate(providers, bindings, task)
		if cand != nil {
			effective[task] = map[string]any{
				"provider_id":    cand.ID,
				"provider_name":  cand.Name,
				"model_name":     cand.Model,
				"interface_mode": cand.InterfaceMode,
				"source":         cand.Source,
			}
		} else {
			effective[task] = map[string]any{}
		}
	}

	return &AIState{
		Providers:    masked,
		TaskBindings: bindings,
		TaskTests:    tests,
		Effective:    effective,
		Summary: map[string]int{
			"provider_count":          len(providers),
			"enabled_provider_count":  enabledCount,
			"detected_provider_count": detected,
		},
		Enabled:     decodeBool(ai["enabled"], false),
		AllowAnon:   decodeBool(ai["allow_anonymous"], false),
		Temperature: decodeFloat(ai["temperature"], 0.7),
		MaxTokens:   decodeInt(ai["max_tokens"], 800),
	}, nil
}

type taskCandidate struct {
	ID            int64
	Name          string
	Model         string
	InterfaceMode string
	APIBaseURL    string
	APIKey        string
	Source        string
}

func (s *SettingsService) resolveTaskCandidate(providers []AIProvider, bindings map[string]AITaskBinding, task string) *taskCandidate {
	b := bindings[task]
	// manual
	if b.Mode == "manual" && b.ProviderID != nil {
		for _, p := range providers {
			if p.ID == *b.ProviderID && p.Enabled {
				model := b.ModelName
				if model == "" && p.RecommendedModels != nil {
					model = p.RecommendedModels[task]
				}
				if model == "" && p.RecommendedModels != nil {
					model = p.RecommendedModels["fallback"]
				}
				return &taskCandidate{
					ID: p.ID, Name: p.Name, Model: model,
					InterfaceMode: p.InterfaceMode, APIBaseURL: p.APIBaseURL, APIKey: p.APIKey,
					Source: "manual",
				}
			}
		}
	}
	// auto: first enabled by priority with recommended or catalog
	sorted := append([]AIProvider{}, providers...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority == sorted[j].Priority {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].Priority < sorted[j].Priority
	})
	for _, p := range sorted {
		if !p.Enabled || strings.TrimSpace(p.APIBaseURL) == "" || strings.TrimSpace(p.APIKey) == "" {
			continue
		}
		model := ""
		if p.RecommendedModels != nil {
			model = p.RecommendedModels[task]
			if model == "" {
				model = p.RecommendedModels["fallback"]
			}
		}
		if model == "" && len(p.ModelCatalog) > 0 {
			model = p.ModelCatalog[0].ID
		}
		return &taskCandidate{
			ID: p.ID, Name: p.Name, Model: model,
			InterfaceMode: p.InterfaceMode, APIBaseURL: p.APIBaseURL, APIKey: p.APIKey,
			Source: "auto",
		}
	}
	return nil
}

// SaveAIProvider creates or updates a provider.
func (s *SettingsService) SaveAIProvider(ctx context.Context, in AIProvider) (*AIState, error) {
	list, err := s.loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = "默认提供方"
	}
	in.APIBaseURL = strings.TrimRight(strings.TrimSpace(in.APIBaseURL), "/")
	if in.APIBaseURL == "" {
		return nil, apperr.New(apperr.Validation, "请填写基础 URL")
	}
	mode := strings.ToLower(strings.TrimSpace(in.InterfaceMode))
	if mode != "chat" && mode != "responses" {
		mode = "auto"
	}
	in.InterfaceMode = mode
	if in.Priority <= 0 {
		in.Priority = 100
	}
	keyIn := strings.TrimSpace(in.APIKey)
	keepKey := keyIn == "" || keyIn == secretSentinel || strings.HasPrefix(keyIn, "****")

	if in.ID > 0 {
		found := false
		for i := range list {
			if list[i].ID != in.ID {
				continue
			}
			found = true
			oldKey := list[i].APIKey
			configChanged := list[i].APIBaseURL != in.APIBaseURL || list[i].InterfaceMode != in.InterfaceMode
			list[i].Name = in.Name
			list[i].APIBaseURL = in.APIBaseURL
			list[i].InterfaceMode = in.InterfaceMode
			list[i].Enabled = in.Enabled
			list[i].Priority = in.Priority
			if keepKey {
				list[i].APIKey = oldKey
			} else {
				list[i].APIKey = keyIn
				configChanged = true
			}
			if configChanged {
				list[i].ModelCatalog = nil
				list[i].RecommendedModels = nil
				list[i].ProbeLastAt = ""
				list[i].ProbeError = ""
				list[i].ProbeSignature = ""
			}
			break
		}
		if !found {
			return nil, apperr.New(apperr.NotFound, "提供方不存在")
		}
	} else {
		if keepKey || keyIn == "" {
			return nil, apperr.New(apperr.Validation, "请填写 API Key")
		}
		var maxID int64
		for _, p := range list {
			if p.ID > maxID {
				maxID = p.ID
			}
		}
		in.ID = maxID + 1
		in.APIKey = keyIn
		list = append(list, in)
	}
	if err := s.saveProviders(ctx, list); err != nil {
		return nil, err
	}
	s.syncLegacyAIFields(ctx, list)
	return s.GetAIState(ctx)
}

func (s *SettingsService) DeleteAIProvider(ctx context.Context, id int64) (*AIState, error) {
	list, err := s.loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	out := list[:0]
	found := false
	for _, p := range list {
		if p.ID == id {
			found = true
			continue
		}
		out = append(out, p)
	}
	if !found {
		return nil, apperr.New(apperr.NotFound, "提供方不存在")
	}
	if err := s.saveProviders(ctx, out); err != nil {
		return nil, err
	}
	// clear bindings pointing to deleted provider
	bindings := s.loadTaskBindings(ctx)
	for k, b := range bindings {
		if b.ProviderID != nil && *b.ProviderID == id {
			b.Mode = "auto"
			b.ProviderID = nil
			b.ModelName = ""
			bindings[k] = b
		}
	}
	_ = s.repo.Set(ctx, "ai", "task_bindings", bindings)
	s.syncLegacyAIFields(ctx, out)
	return s.GetAIState(ctx)
}

func (s *SettingsService) syncLegacyAIFields(ctx context.Context, list []AIProvider) {
	// keep first enabled as legacy single-line for embedding fallback
	sorted := append([]AIProvider{}, list...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	var primary *AIProvider
	for i := range sorted {
		if sorted[i].Enabled {
			primary = &sorted[i]
			break
		}
	}
	if primary == nil && len(sorted) > 0 {
		primary = &sorted[0]
	}
	if primary == nil {
		return
	}
	_ = s.repo.SetNamespace(ctx, "ai", map[string]any{
		"api_base_url":   primary.APIBaseURL,
		"api_key":        primary.APIKey,
		"interface_mode": primary.InterfaceMode,
	})
}

func (s *SettingsService) SaveTaskBindings(ctx context.Context, bindings map[string]AITaskBinding) (*AIState, error) {
	out := defaultTaskBindings()
	for _, k := range AITaskKeys {
		if b, ok := bindings[k]; ok {
			if b.Mode != "manual" {
				b.Mode = "auto"
			}
			out[k] = b
		}
	}
	if err := s.repo.Set(ctx, "ai", "task_bindings", out); err != nil {
		return nil, err
	}
	return s.GetAIState(ctx)
}

// DetectProviderModels lists /v1/models and stores catalog + naive recommendations.
func (s *SettingsService) DetectProviderModels(ctx context.Context, id int64) (*AIState, error) {
	list, err := s.loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	idx := -1
	for i := range list {
		if list[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, apperr.New(apperr.NotFound, "提供方不存在")
	}
	p := &list[idx]
	if strings.TrimSpace(p.APIBaseURL) == "" || strings.TrimSpace(p.APIKey) == "" {
		return nil, apperr.New(apperr.Validation, "请先保存完整的基础 URL 和密钥")
	}
	catalog, err := listOpenAIModels(ctx, p.APIBaseURL, p.APIKey)
	if err != nil {
		p.ProbeError = err.Error()
		p.ProbeLastAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.saveProviders(ctx, list)
		return nil, apperr.New(apperr.Validation, "模型探测失败: "+err.Error())
	}
	p.ModelCatalog = catalog
	p.RecommendedModels = recommendModels(catalog)
	p.ProbeLastAt = time.Now().UTC().Format(time.RFC3339)
	p.ProbeError = ""
	p.ProbeSignature = fmt.Sprintf("%s|%d", p.APIBaseURL, len(catalog))
	if err := s.saveProviders(ctx, list); err != nil {
		return nil, err
	}
	return s.GetAIState(ctx)
}

func (s *SettingsService) DetectAllProviders(ctx context.Context) (*AIState, []string, []string, error) {
	list, err := s.loadProviders(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	var okNames, errs []string
	for i := range list {
		p := &list[i]
		if !p.Enabled {
			continue
		}
		if strings.TrimSpace(p.APIBaseURL) == "" || strings.TrimSpace(p.APIKey) == "" {
			errs = append(errs, p.Name+": 缺少 URL 或密钥")
			continue
		}
		catalog, err := listOpenAIModels(ctx, p.APIBaseURL, p.APIKey)
		if err != nil {
			p.ProbeError = err.Error()
			p.ProbeLastAt = time.Now().UTC().Format(time.RFC3339)
			errs = append(errs, p.Name+": "+err.Error())
			continue
		}
		p.ModelCatalog = catalog
		p.RecommendedModels = recommendModels(catalog)
		p.ProbeLastAt = time.Now().UTC().Format(time.RFC3339)
		p.ProbeError = ""
		p.ProbeSignature = fmt.Sprintf("%s|%d", p.APIBaseURL, len(catalog))
		okNames = append(okNames, p.Name)
	}
	if err := s.saveProviders(ctx, list); err != nil {
		return nil, nil, nil, err
	}
	state, err := s.GetAIState(ctx)
	return state, okNames, errs, err
}

func (s *SettingsService) TestProvider(ctx context.Context, id int64, modelName string) (map[string]any, error) {
	list, err := s.loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	var p *AIProvider
	for i := range list {
		if list[i].ID == id {
			p = &list[i]
			break
		}
	}
	if p == nil {
		return nil, apperr.New(apperr.NotFound, "提供方不存在")
	}
	if modelName == "" {
		if p.RecommendedModels != nil {
			modelName = p.RecommendedModels["fallback"]
			if modelName == "" {
				modelName = p.RecommendedModels["intent"]
			}
		}
		if modelName == "" && len(p.ModelCatalog) > 0 {
			modelName = p.ModelCatalog[0].ID
		}
	}
	if modelName == "" {
		return nil, apperr.New(apperr.Validation, "请先检测模型或指定模型名")
	}
	proto, err := probeChat(ctx, p.APIBaseURL, p.APIKey, modelName, "ping")
	if err != nil {
		return nil, apperr.New(apperr.Validation, "测试失败: "+err.Error())
	}
	return map[string]any{
		"ok":      true,
		"message": fmt.Sprintf("提供方测试成功：%s / %s", p.Name, modelName),
		"details": map[string]any{"transport_protocol": proto, "tested_model": modelName},
	}, nil
}

func (s *SettingsService) TestAllTasks(ctx context.Context) (*AIState, error) {
	list, err := s.loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	bindings := s.loadTaskBindings(ctx)
	results := defaultTaskTestResults()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, task := range AITaskKeys {
		cand := s.resolveTaskCandidate(list, bindings, task)
		if cand == nil || cand.Model == "" {
			results[task] = AITaskTestResult{
				Status: "error", Message: "未找到可用的提供方或模型", TestedAt: now,
			}
			continue
		}
		// lightweight structured probe: chat completion with JSON instruction
		prompt := taskProbePrompt(task)
		proto, err := probeChat(ctx, cand.APIBaseURL, cand.APIKey, cand.Model, prompt)
		pid := cand.ID
		if err != nil {
			results[task] = AITaskTestResult{
				Status: "error", Message: err.Error(), ProviderID: &pid,
				ProviderName: cand.Name, ModelName: cand.Model, TestedAt: now, Protocol: proto,
			}
			continue
		}
		results[task] = AITaskTestResult{
			Status: "success", Message: AITaskLabels[task] + "测试通过",
			ProviderID: &pid, ProviderName: cand.Name, ModelName: cand.Model,
			TestedAt: now, Protocol: proto,
		}
	}
	if err := s.repo.Set(ctx, "ai", "task_test_results", results); err != nil {
		return nil, err
	}
	return s.GetAIState(ctx)
}

func taskProbePrompt(task string) string {
	switch task {
	case "intent":
		return `请用一行 JSON 回复：{"intent":"search","keywords":["AI"]}`
	case "rerank":
		return `请用一行 JSON 回复：{"order":[1,2,3]}`
	case "translate":
		return `请把「导航」翻译成英文，只返回英文单词`
	case "site_info":
		return `请用一行 JSON 回复：{"title":"Example","description":"demo"}`
	default:
		return "ping"
	}
}

func listOpenAIModels(ctx context.Context, baseURL, apiKey string) ([]AIModelInfo, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s: %s", resp.Status, truncate(string(b), 200))
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("解析模型列表失败")
	}
	var catalog []AIModelInfo
	for _, m := range out.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		compat := "unknown"
		low := strings.ToLower(id)
		if strings.Contains(low, "gpt") || strings.Contains(low, "claude") || strings.Contains(low, "qwen") ||
			strings.Contains(low, "deepseek") || strings.Contains(low, "gemini") || strings.Contains(low, "chat") {
			compat = "full"
		}
		if strings.Contains(low, "embed") {
			compat = "partial"
		}
		catalog = append(catalog, AIModelInfo{ID: id, Compatible: compat})
	}
	if len(catalog) == 0 {
		return nil, fmt.Errorf("未发现任何模型")
	}
	// cap catalog size for storage
	if len(catalog) > 200 {
		catalog = catalog[:200]
	}
	return catalog, nil
}

func recommendModels(catalog []AIModelInfo) map[string]string {
	pick := func(prefer ...string) string {
		for _, p := range prefer {
			pl := strings.ToLower(p)
			for _, m := range catalog {
				if strings.Contains(strings.ToLower(m.ID), pl) && m.Compatible != "partial" {
					return m.ID
				}
			}
		}
		for _, m := range catalog {
			if m.Compatible == "full" {
				return m.ID
			}
		}
		if len(catalog) > 0 {
			return catalog[0].ID
		}
		return ""
	}
	fallback := pick("gpt-4o-mini", "gpt-4o", "gpt-3.5", "qwen", "deepseek", "claude", "gemini", "chat")
	return map[string]string{
		"intent":    fallback,
		"rerank":    fallback,
		"translate": fallback,
		"site_info": fallback,
		"fallback":  fallback,
	}
}

func probeChat(ctx context.Context, baseURL, apiKey, model, userMsg string) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": userMsg},
		},
		"max_tokens":  64,
		"temperature": 0,
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
		return "chat", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return "chat", fmt.Errorf("HTTP %s: %s", resp.Status, truncate(string(raw), 200))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "chat", fmt.Errorf("解析响应失败")
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "chat", fmt.Errorf("模型返回空内容")
	}
	return "chat", nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
