package service

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/vector"
	"github.com/booknav/book-nav/apps/server/internal/repository"
)

// secretSentinel is returned to admin UI instead of real API keys.
const secretSentinel = "********"

type SettingsService struct {
	repo *repository.SettingsRepo
}

func NewSettingsService(repo *repository.SettingsRepo) *SettingsService {
	return &SettingsService{repo: repo}
}

func (s *SettingsService) EnsureDefaults(ctx context.Context) error {
	// Qdrant 默认：环境变量优先，便于容器接公共/共享服务
	qdrantDefault := strings.TrimSpace(os.Getenv("BOOKNAV_QDRANT_URL"))
	if qdrantDefault == "" {
		qdrantDefault = strings.TrimSpace(os.Getenv("QDRANT_URL"))
	}
	if qdrantDefault == "" {
		qdrantDefault = "http://localhost:6333"
	}

	defaults := map[string]map[string]any{
		"site": {
			"name":        "BookNav",
			"subtitle":    "",
			"logo":        "",
			"favicon":     "",
			"footer":      "",
			"keywords":    "",
			"description": "",
		},
		"transition": {
			"enable":           false,
			"time":             5,
			"admin_time":       0,
			"remember_choice":  true,
			"show_description": true,
			"theme":            "default",
			"color":            "#6e8efb",
			"ad1":              "",
			"ad2":              "",
		},
		// WebDAV 云备份配置列表（备份管理页）
		"webdav": {
			"configs": []any{},
		},
		// AI 搜索（多 Provider + 四任务绑定见 ai.providers / task_bindings）
		"ai": {
			"enabled":         false,
			"allow_anonymous": false,
			"api_base_url":    "",
			"api_key":         "",
			"model":           "",
			"interface_mode":  "auto",
			"temperature":     0.7,
			"max_tokens":      800,
			"providers":       []any{},
			"task_bindings": map[string]any{
				"intent":    map[string]any{"mode": "auto", "provider_id": nil, "model_name": ""},
				"rerank":    map[string]any{"mode": "auto", "provider_id": nil, "model_name": ""},
				"translate": map[string]any{"mode": "auto", "provider_id": nil, "model_name": ""},
				"site_info": map[string]any{"mode": "auto", "provider_id": nil, "model_name": ""},
			},
			"task_test_results": map[string]any{},
		},
		// 向量 + Qdrant（前台 AI 搜索核心召回）
		"vector": {
			"enabled":                false,
			"qdrant_url":             qdrantDefault,
			"collection":             "websites",
			"embedding_api_base_url": "",
			"embedding_api_key":      "",
			"embedding_model":        "text-embedding-3-small",
			"dimension":              1536,
			"similarity_threshold":   0.3,
			"max_results":            50,
		},
		"announcement": {
			"enabled":       false,
			"title":         "",
			"content":       "",
			"start":         "",
			"end":           "",
			"remember_days": 7,
		},
		"icon": {
			"display_mode":      "smart",
			"auto_fetch":        true,
			"sync_local":        true,
			"sync_imagebed":     false,
			"imagebed_provider": "",
			"imagebed_api_url":  "",
			"imagebed_token":    "",
			"source_providers":  defaultIconSourceProviders(),
		},
	}

	for ns, kv := range defaults {
		existing, err := s.repo.GetNamespace(ctx, ns)
		if err != nil {
			return err
		}
		toSet := map[string]any{}
		for k, v := range kv {
			if _, ok := existing[k]; !ok {
				toSet[k] = v
			}
		}
		if len(toSet) > 0 {
			if err := s.repo.SetNamespace(ctx, ns, toSet); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SettingsService) Public(ctx context.Context) (domain.PublicSettings, error) {
	site, _ := s.repo.GetNamespace(ctx, "site")
	tr, _ := s.repo.GetNamespace(ctx, "transition")
	ai, _ := s.repo.GetNamespace(ctx, "ai")
	ann, _ := s.repo.GetNamespace(ctx, "announcement")
	vec, _ := s.repo.GetNamespace(ctx, "vector")

	// 前台「AI 搜索」：总开关或向量启用即可
	aiOn := decodeBool(ai["enabled"], false)
	vectorOn := decodeBool(vec["enabled"], false)

	return domain.PublicSettings{
		SiteName:             decodeStr(site["name"], "BookNav"),
		SiteSubtitle:         decodeStr(site["subtitle"], ""),
		SiteLogo:             decodeStr(site["logo"], ""),
		SiteFavicon:          decodeStr(site["favicon"], ""),
		FooterContent:        decodeStr(site["footer"], ""),
		AISearchEnabled:      aiOn || vectorOn,
		AIAllowAnon:          decodeBool(ai["allow_anonymous"], false),
		EnableTransition:     decodeBool(tr["enable"], false),
		TransitionTime:       decodeInt(tr["time"], 5),
		AdminTransition:      decodeInt(tr["admin_time"], 0),
		TransitionRemember:   decodeBool(tr["remember_choice"], true),
		TransitionShowDesc:   decodeBool(tr["show_description"], true),
		TransitionTheme:      decodeStr(tr["theme"], "default"),
		TransitionColor:      decodeStr(tr["color"], "#6e8efb"),
		TransitionAd1:        decodeStr(tr["ad1"], ""),
		TransitionAd2:        decodeStr(tr["ad2"], ""),
		AnnouncementOn:       decodeBool(ann["enabled"], false),
		AnnouncementTitle:    decodeStr(ann["title"], ""),
		AnnouncementContent:  decodeStr(ann["content"], ""),
		AnnouncementStart:    decodeStr(ann["start"], ""),
		AnnouncementEnd:      decodeStr(ann["end"], ""),
		AnnouncementRemember: decodeInt(ann["remember_days"], 7),
	}, nil
}

// GetNamespaceForAdmin masks secrets for admin UI.
func (s *SettingsService) GetNamespaceForAdmin(ctx context.Context, ns string) (map[string]any, error) {
	raw, err := s.repo.GetNamespace(ctx, ns)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range raw {
		var decoded any
		if err := json.Unmarshal(v, &decoded); err != nil {
			out[k] = string(v)
			continue
		}
		if isSecretKey(k) {
			if str, ok := decoded.(string); ok && strings.TrimSpace(str) != "" {
				out[k] = secretSentinel
				out[k+"_configured"] = true
			} else {
				out[k] = ""
				out[k+"_configured"] = false
			}
			continue
		}
		out[k] = decoded
	}
	// icon: always return merged source providers for UI
	if ns == "icon" {
		out["source_providers"] = mergeIconSourceProviders(out["source_providers"])
	}
	return out, nil
}

func (s *SettingsService) GetNamespace(ctx context.Context, ns string) (map[string]json.RawMessage, error) {
	return s.repo.GetNamespace(ctx, ns)
}

// SetNamespace merges values; empty/sentinel secrets keep previous values.
func (s *SettingsService) SetNamespace(ctx context.Context, ns string, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	existing, err := s.repo.GetNamespace(ctx, ns)
	if err != nil {
		return err
	}
	cleaned := map[string]any{}
	for k, v := range values {
		if strings.HasSuffix(k, "_configured") {
			continue
		}
		if isSecretKey(k) {
			str, _ := v.(string)
			str = strings.TrimSpace(str)
			if str == "" || str == secretSentinel || strings.HasPrefix(str, "****") {
				if prev, ok := existing[k]; ok && len(prev) > 0 {
					var prevStr string
					if json.Unmarshal(prev, &prevStr) == nil {
						cleaned[k] = prevStr
					}
				}
				continue
			}
		}
		cleaned[k] = v
	}
	if len(cleaned) == 0 {
		return nil
	}
	return s.repo.SetNamespace(ctx, ns, cleaned)
}

// VectorConfig builds vector.Client config from settings + env overrides.
func (s *SettingsService) VectorConfig(ctx context.Context) (vector.Config, bool) {
	vec, _ := s.repo.GetNamespace(ctx, "vector")
	ai, _ := s.repo.GetNamespace(ctx, "ai")

	enabled := decodeBool(vec["enabled"], false)
	qdrant := decodeStr(vec["qdrant_url"], "")
	if env := strings.TrimSpace(os.Getenv("BOOKNAV_QDRANT_URL")); env != "" {
		qdrant = env
	} else if env := strings.TrimSpace(os.Getenv("QDRANT_URL")); env != "" {
		qdrant = env
	}

	embedURL := decodeStr(vec["embedding_api_base_url"], "")
	embedKey := decodeStr(vec["embedding_api_key"], "")
	// 留空则回落到 AI 搜索 API
	if embedURL == "" {
		embedURL = decodeStr(ai["api_base_url"], "")
	}
	if embedKey == "" {
		embedKey = decodeStr(ai["api_key"], "")
	}

	cfg := vector.Config{
		QdrantURL:      qdrant,
		Collection:     decodeStr(vec["collection"], "websites"),
		EmbeddingURL:   embedURL,
		EmbeddingKey:   embedKey,
		EmbeddingModel: decodeStr(vec["embedding_model"], "text-embedding-3-small"),
		Dimension:      decodeInt(vec["dimension"], 1536),
		Threshold:      decodeFloat(vec["similarity_threshold"], 0.3),
		MaxResults:     decodeInt(vec["max_results"], 50),
	}.Normalize()

	ready := enabled && cfg.QdrantURL != "" && cfg.EmbeddingURL != "" && cfg.EmbeddingKey != ""
	return cfg, ready
}

// AIEnabled reports whether AI/vector search may be offered.
func (s *SettingsService) AIEnabled(ctx context.Context) (enabled, allowAnon bool) {
	ai, _ := s.repo.GetNamespace(ctx, "ai")
	vec, _ := s.repo.GetNamespace(ctx, "vector")
	enabled = decodeBool(ai["enabled"], false) || decodeBool(vec["enabled"], false)
	allowAnon = decodeBool(ai["allow_anonymous"], false)
	return
}

func isSecretKey(k string) bool {
	k = strings.ToLower(k)
	return k == "api_key" || k == "embedding_api_key" || k == "imagebed_token" ||
		strings.HasSuffix(k, "_api_key") || strings.HasSuffix(k, "_token") || k == "password"
}

func decodeStr(raw json.RawMessage, def string) string {
	if len(raw) == 0 {
		return def
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return def
	}
	return s
}

func decodeBool(raw json.RawMessage, def bool) bool {
	if len(raw) == 0 {
		return def
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return def
	}
	return b
}

func decodeInt(raw json.RawMessage, def int) int {
	if len(raw) == 0 {
		return def
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		// try float (JSON numbers from JS)
		var f float64
		if err2 := json.Unmarshal(raw, &f); err2 != nil {
			return def
		}
		return int(f)
	}
	return n
}

func decodeFloat(raw json.RawMessage, def float64) float64 {
	if len(raw) == 0 {
		return def
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return def
	}
	return f
}
