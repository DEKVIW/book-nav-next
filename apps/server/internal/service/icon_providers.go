package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// IconSourceProvider describes a favicon source (origin HTML parse or proxy template).
type IconSourceProvider struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Kind             string `json:"kind"` // origin | proxy
	Builtin          bool   `json:"builtin"`
	Enabled          bool   `json:"enabled"`
	Order            int    `json:"order"`
	SupportsDownload bool   `json:"supports_download"`
	Description      string `json:"description"`
	Template         string `json:"template,omitempty"`
}

func defaultIconSourceProviders() []IconSourceProvider {
	return []IconSourceProvider{
		{
			ID: "origin_direct", Label: "原站直连", Kind: "origin", Builtin: true,
			Enabled: true, Order: 10, SupportsDownload: true,
			Description: "优先解析网站自身声明的 favicon 与常见图标路径",
		},
		{
			ID: "favicon_im", Label: "favicon.im", Kind: "proxy", Builtin: true,
			Enabled: true, Order: 20, SupportsDownload: true,
			Description: "Cloudflare 加速代理",
			Template:    "https://favicon.im/{domain}?larger=true",
		},
		{
			ID: "vemetric", Label: "Vemetric", Kind: "proxy", Builtin: true,
			Enabled: true, Order: 30, SupportsDownload: true,
			Description: "支持尺寸与格式控制",
			Template:    "https://favicon.vemetric.com/{domain}?size={size}&format=png",
		},
		{
			ID: "google_s2", Label: "Google S2", Kind: "proxy", Builtin: true,
			Enabled: true, Order: 40, SupportsDownload: true,
			Description: "经典稳定代理",
			Template:    "https://www.google.com/s2/favicons?domain={domain}&sz={size}",
		},
		{
			ID: "duckduckgo", Label: "DuckDuckGo", Kind: "proxy", Builtin: true,
			Enabled: true, Order: 50, SupportsDownload: true,
			Description: "隐私友好代理",
			Template:    "https://icons.duckduckgo.com/ip3/{domain}.ico",
		},
		{
			ID: "cccyun", Label: "CCCYun", Kind: "proxy", Builtin: true,
			Enabled: false, Order: 60, SupportsDownload: true,
			Description: "旧兼容代理，末位兜底",
			Template:    "https://favicon.cccyun.cc/{domain}",
		},
	}
}

// mergeIconSourceProviders merges stored JSON config with built-in defaults.
func mergeIconSourceProviders(raw any) []IconSourceProvider {
	defaults := defaultIconSourceProviders()
	byID := map[string]IconSourceProvider{}
	for _, p := range defaults {
		byID[p.ID] = p
	}

	var stored []map[string]any
	switch v := raw.(type) {
	case nil:
		// use defaults
	case []IconSourceProvider:
		// already typed
		out := append([]IconSourceProvider{}, v...)
		sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
		return out
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				stored = append(stored, m)
			}
		}
	case string:
		if strings.TrimSpace(v) != "" {
			_ = json.Unmarshal([]byte(v), &stored)
		}
	case json.RawMessage:
		if len(v) > 0 {
			_ = json.Unmarshal(v, &stored)
		}
	case []byte:
		if len(v) > 0 {
			_ = json.Unmarshal(v, &stored)
		}
	default:
		// try re-marshal
		b, err := json.Marshal(v)
		if err == nil {
			_ = json.Unmarshal(b, &stored)
		}
	}

	// apply stored enabled/order/template overrides by id
	seen := map[string]bool{}
	var out []IconSourceProvider
	for _, m := range stored {
		id, _ := m["id"].(string)
		if id == "" {
			continue
		}
		base, ok := byID[id]
		if !ok {
			// custom provider
			base = IconSourceProvider{ID: id, Builtin: false, Kind: "proxy", SupportsDownload: true}
		}
		if lab, ok := m["label"].(string); ok && lab != "" {
			base.Label = lab
		}
		if kind, ok := m["kind"].(string); ok && kind != "" {
			base.Kind = kind
		}
		if en, ok := m["enabled"].(bool); ok {
			base.Enabled = en
		}
		if ord, ok := m["order"].(float64); ok {
			base.Order = int(ord)
		} else if ord, ok := m["order"].(int); ok {
			base.Order = ord
		}
		if tpl, ok := m["template"].(string); ok && tpl != "" {
			base.Template = tpl
		}
		if desc, ok := m["description"].(string); ok {
			base.Description = desc
		}
		out = append(out, base)
		seen[id] = true
	}
	// append missing builtins
	for _, p := range defaults {
		if !seen[p.ID] {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

func enabledIconProviders(list []IconSourceProvider) []IconSourceProvider {
	var out []IconSourceProvider
	for _, p := range list {
		if p.Enabled {
			out = append(out, p)
		}
	}
	return out
}

func domainFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		// try add scheme
		u2, err2 := url.Parse("https://" + strings.TrimSpace(raw))
		if err2 != nil {
			return ""
		}
		return strings.ToLower(u2.Hostname())
	}
	return strings.ToLower(u.Hostname())
}

func expandIconTemplate(tpl, domain string, size int) string {
	if size <= 0 {
		size = 64
	}
	r := strings.NewReplacer(
		"{domain}", domain,
		"{host}", domain,
		"{size}", fmt.Sprintf("%d", size),
	)
	return r.Replace(tpl)
}

// discoverIconWithProviders tries origin HTML parse then enabled proxy templates.
func (s *JobService) discoverIconWithProviders(pageURL string, providers []IconSourceProvider) (string, error) {
	list := enabledIconProviders(providers)
	if len(list) == 0 {
		list = enabledIconProviders(defaultIconSourceProviders())
	}
	domain := domainFromURL(pageURL)
	for _, p := range list {
		switch p.Kind {
		case "origin", "":
			if p.ID == "origin_direct" || p.Kind == "origin" {
				if u, err := s.discoverIconFromPage(pageURL); err == nil && u != "" {
					return u, nil
				}
			}
		case "proxy":
			if domain == "" || p.Template == "" {
				continue
			}
			u := expandIconTemplate(p.Template, domain, 64)
			if s.probeIconURL(u) {
				return u, nil
			}
		}
	}
	// last resort
	return fallbackFavicon(pageURL), nil
}

func (s *JobService) discoverIconFromPage(pageURL string) (string, error) {
	return s.discoverIcon(pageURL)
}

func (s *JobService) probeIconURL(iconURL string) bool {
	if err := assertPublicURL(iconURL); err != nil {
		return false
	}
	req, err := http.NewRequest(http.MethodHead, iconURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "BookNav-IconFetcher/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		// some CDNs block HEAD — try GET range lightly
		req, err = http.NewRequest(http.MethodGet, iconURL, nil)
		if err != nil {
			return false
		}
		req.Header.Set("User-Agent", "BookNav-IconFetcher/1.0")
		req.Header.Set("Range", "bytes=0-0")
		resp, err = s.httpClient.Do(req)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
