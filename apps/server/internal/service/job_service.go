package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
	"github.com/booknav/book-nav/apps/server/internal/pkg/vector"
	"github.com/booknav/book-nav/apps/server/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)

type JobService struct {
	jobs       *repository.JobRepo
	websites   *repository.WebsiteRepo
	deadlinks  *repository.DeadlinkRepo
	settings   *SettingsService
	dataDir    string
	httpClient *http.Client
}

func NewJobService(
	jobs *repository.JobRepo,
	websites *repository.WebsiteRepo,
	deadlinks *repository.DeadlinkRepo,
	settings *SettingsService,
	dataDir string,
) *JobService {
	return &JobService{
		jobs:      jobs,
		websites:  websites,
		deadlinks: deadlinks,
		settings:  settings,
		dataDir:   dataDir,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (s *JobService) Get(ctx context.Context, id int64) (*domain.Job, error) {
	j, err := s.jobs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if j == nil {
		return nil, apperr.New(apperr.NotFound, "任务不存在")
	}
	return j, nil
}

func (s *JobService) List(ctx context.Context) ([]domain.Job, error) {
	return s.jobs.List(ctx, 50)
}

func (s *JobService) Delete(ctx context.Context, id int64) error {
	j, err := s.jobs.Get(ctx, id)
	if err != nil {
		return err
	}
	if j == nil {
		return apperr.New(apperr.NotFound, "任务不存在")
	}
	if j.Status == "pending" || j.Status == "running" {
		return apperr.New(apperr.Validation, "运行中的任务无法删除")
	}
	return s.jobs.Delete(ctx, id)
}

func (s *JobService) ClearFinished(ctx context.Context) (int64, error) {
	return s.jobs.DeleteFinished(ctx)
}

func (s *JobService) StartDeadlinkCheck(ctx context.Context, user *domain.User) (*domain.Job, error) {
	if user == nil || !user.Role.IsSuperAdmin() {
		return nil, apperr.New(apperr.Forbidden, "需要超级管理员")
	}
	uid := user.ID
	batch := uuid.NewString()
	payload, _ := json.Marshal(map[string]string{"batch_id": batch})
	j := &domain.Job{
		Type:        "deadlink_check",
		Status:      "pending",
		PayloadJSON: string(payload),
		CreatedBy:   &uid,
		ResultJSON:  "{}",
	}
	if err := s.jobs.Create(ctx, j); err != nil {
		return nil, err
	}
	go s.runDeadlink(j.ID, batch)
	return j, nil
}

func (s *JobService) StartIconFetch(ctx context.Context, user *domain.User) (*domain.Job, error) {
	if user == nil || !user.Role.IsSuperAdmin() {
		return nil, apperr.New(apperr.Forbidden, "需要超级管理员")
	}
	uid := user.ID
	j := &domain.Job{
		Type:        "icon_sync",
		Status:      "pending",
		PayloadJSON: "{}",
		CreatedBy:   &uid,
		ResultJSON:  "{}",
	}
	if err := s.jobs.Create(ctx, j); err != nil {
		return nil, err
	}
	go s.runIconFetch(j.ID)
	return j, nil
}

// StartVectorIndex bulk-embeds all websites into Qdrant.
func (s *JobService) StartVectorIndex(ctx context.Context, user *domain.User) (*domain.Job, error) {
	if user == nil || !user.Role.IsSuperAdmin() {
		return nil, apperr.New(apperr.Forbidden, "需要超级管理员")
	}
	if s.settings == nil {
		return nil, apperr.New(apperr.Validation, "设置服务不可用")
	}
	cfg, ready := s.settings.VectorConfig(ctx)
	if !ready {
		return nil, apperr.New(apperr.Validation, "向量搜索未配置完整：请启用并填写 Qdrant 与 Embedding API")
	}
	// quick connectivity check
	client := vector.NewClient(cfg)
	if err := client.PingQdrant(ctx); err != nil {
		return nil, apperr.New(apperr.Validation, "Qdrant 连接失败: "+err.Error())
	}
	uid := user.ID
	j := &domain.Job{
		Type:        "vector_index",
		Status:      "pending",
		PayloadJSON: "{}",
		CreatedBy:   &uid,
		ResultJSON:  "{}",
	}
	if err := s.jobs.Create(ctx, j); err != nil {
		return nil, err
	}
	go s.runVectorIndex(j.ID)
	return j, nil
}

// TestVectorConfig pings embedding + qdrant using current settings (or draft overrides).
func (s *JobService) TestVectorConfig(ctx context.Context, overrides map[string]any) (map[string]any, error) {
	if s.settings == nil {
		return nil, apperr.New(apperr.Validation, "设置服务不可用")
	}
	cfg, _ := s.settings.VectorConfig(ctx)
	// apply draft overrides from UI without saving
	if v, ok := overrides["qdrant_url"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.QdrantURL = strings.TrimRight(strings.TrimSpace(v), "/")
	}
	if v, ok := overrides["embedding_api_base_url"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.EmbeddingURL = strings.TrimRight(strings.TrimSpace(v), "/")
	}
	if v, ok := overrides["embedding_api_key"].(string); ok {
		key := strings.TrimSpace(v)
		if key != "" && key != "********" && !strings.HasPrefix(key, "****") {
			cfg.EmbeddingKey = key
		}
	}
	if v, ok := overrides["embedding_model"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.EmbeddingModel = strings.TrimSpace(v)
	}
	if v, ok := overrides["collection"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.Collection = strings.TrimSpace(v)
	}
	cfg = cfg.Normalize()
	client := vector.NewClient(cfg)

	details := map[string]string{}
	ok := true
	if err := client.PingQdrant(ctx); err != nil {
		details["qdrant"] = "失败: " + err.Error()
		ok = false
	} else {
		details["qdrant"] = "连接成功"
	}
	if err := client.PingEmbedding(ctx); err != nil {
		details["embedding_api"] = "失败: " + err.Error()
		ok = false
	} else {
		details["embedding_api"] = "连接成功"
	}
	msg := "向量搜索连接测试完成"
	if !ok {
		msg = "向量搜索连接测试存在失败项"
	}
	return map[string]any{
		"ok":      ok,
		"message": msg,
		"details": details,
		"config": map[string]any{
			"qdrant_url":      cfg.QdrantURL,
			"collection":      cfg.Collection,
			"embedding_model": cfg.EmbeddingModel,
			"dimension":       cfg.Dimension,
		},
	}, nil
}

func (s *JobService) runVectorIndex(jobID int64) {
	ctx := context.Background()
	j, err := s.jobs.Get(ctx, jobID)
	if err != nil || j == nil {
		return
	}
	now := clock.NowUTC()
	j.Status = "running"
	j.StartedAt = &now
	_ = s.jobs.Update(ctx, j)

	cfg, ready := s.settings.VectorConfig(ctx)
	if !ready {
		j.Status = "failed"
		j.Error = "vector config incomplete"
		fin := clock.NowUTC()
		j.FinishedAt = &fin
		_ = s.jobs.Update(ctx, j)
		return
	}
	client := vector.NewClient(cfg)
	if err := client.EnsureCollection(ctx); err != nil {
		j.Status = "failed"
		j.Error = "ensure collection: " + err.Error()
		fin := clock.NowUTC()
		j.FinishedAt = &fin
		_ = s.jobs.Update(ctx, j)
		return
	}

	sites, err := s.websites.ListAll(ctx)
	if err != nil {
		j.Status = "failed"
		j.Error = err.Error()
		fin := clock.NowUTC()
		j.FinishedAt = &fin
		_ = s.jobs.Update(ctx, j)
		return
	}
	j.Total = len(sites)
	_ = s.jobs.Update(ctx, j)

	for i, site := range sites {
		text := buildWebsiteEmbedText(site)
		emb, err := client.Embed(ctx, text)
		if err != nil {
			j.Failed++
			j.Progress = i + 1
			_ = s.jobs.Update(ctx, j)
			continue
		}
		payload := map[string]any{
			"title":       site.Title,
			"url":         site.URL,
			"description": site.Description,
			"category":    site.CategoryName,
		}
		if err := client.UpsertPoint(ctx, site.ID, emb, payload); err != nil {
			j.Failed++
		} else {
			j.Success++
		}
		j.Progress = i + 1
		_ = s.jobs.Update(ctx, j)
	}
	fin := clock.NowUTC()
	j.FinishedAt = &fin
	j.Status = "completed"
	_ = s.jobs.Update(ctx, j)
}

func buildWebsiteEmbedText(w domain.Website) string {
	parts := []string{w.Title, w.Description, w.URL, w.CategoryName}
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(p)
	}
	if b.Len() == 0 {
		return w.URL
	}
	return b.String()
}

func (s *JobService) runDeadlink(jobID int64, batch string) {
	ctx := context.Background()
	j, err := s.jobs.Get(ctx, jobID)
	if err != nil || j == nil {
		return
	}
	now := clock.NowUTC()
	j.Status = "running"
	j.StartedAt = &now
	_ = s.jobs.Update(ctx, j)

	sites, err := s.websites.ListAll(ctx)
	if err != nil {
		j.Status = "failed"
		j.Error = err.Error()
		fin := clock.NowUTC()
		j.FinishedAt = &fin
		_ = s.jobs.Update(ctx, j)
		return
	}
	j.Total = len(sites)
	_ = s.jobs.Update(ctx, j)

	for i, site := range sites {
		valid, code, errType, errMsg, ms := s.checkURL(site.URL)
		sc := code
		rt := ms
		d := &domain.DeadlinkCheck{
			BatchID:        batch,
			WebsiteID:      site.ID,
			URL:            site.URL,
			IsValid:        valid,
			ErrorType:      errType,
			ErrorMessage:   errMsg,
			ResponseTimeMs: &rt,
		}
		if code > 0 {
			d.StatusCode = &sc
		}
		_ = s.deadlinks.Create(ctx, d)
		_ = s.websites.UpdateValidity(ctx, site.ID, valid, clock.NowRFC3339())
		if valid {
			j.Success++
		} else {
			j.Failed++
		}
		j.Progress = i + 1
		_ = s.jobs.Update(ctx, j)
	}
	fin := clock.NowUTC()
	j.FinishedAt = &fin
	j.Status = "completed"
	result, _ := json.Marshal(map[string]any{"batch_id": batch})
	j.ResultJSON = string(result)
	_ = s.jobs.Update(ctx, j)
}

func (s *JobService) runIconFetch(jobID int64) {
	ctx := context.Background()
	j, err := s.jobs.Get(ctx, jobID)
	if err != nil || j == nil {
		return
	}
	now := clock.NowUTC()
	j.Status = "running"
	j.StartedAt = &now
	_ = s.jobs.Update(ctx, j)

	sites, err := s.websites.ListAll(ctx)
	if err != nil {
		j.Status = "failed"
		j.Error = err.Error()
		fin := clock.NowUTC()
		j.FinishedAt = &fin
		_ = s.jobs.Update(ctx, j)
		return
	}
	j.Total = len(sites)
	iconDir := filepath.Join(s.dataDir, "uploads", "icons")
	_ = os.MkdirAll(iconDir, 0o755)

	for i, site := range sites {
		if strings.TrimSpace(site.Icon) != "" && !strings.HasPrefix(site.Icon, "http") {
			j.Success++
			j.Progress = i + 1
			_ = s.jobs.Update(ctx, j)
			continue
		}
		iconURL, err := s.discoverIcon(site.URL)
		if err != nil || iconURL == "" {
			j.Failed++
			j.Progress = i + 1
			_ = s.jobs.Update(ctx, j)
			continue
		}
		local, err := s.downloadIcon(iconURL, iconDir)
		if err != nil {
			// keep remote url
			site.Icon = iconURL
			_ = s.websites.Update(ctx, &site)
			j.Success++
		} else {
			site.Icon = "/media/icons/" + filepath.Base(local)
			_ = s.websites.Update(ctx, &site)
			j.Success++
		}
		j.Progress = i + 1
		_ = s.jobs.Update(ctx, j)
	}
	fin := clock.NowUTC()
	j.FinishedAt = &fin
	j.Status = "completed"
	_ = s.jobs.Update(ctx, j)
}

func (s *JobService) checkURL(raw string) (valid bool, status int, errType, errMsg string, ms int) {
	start := time.Now()
	if err := assertPublicURL(raw); err != nil {
		return false, 0, "ssrf", err.Error(), 0
	}
	req, err := http.NewRequest(http.MethodHead, raw, nil)
	if err != nil {
		return false, 0, "request", err.Error(), 0
	}
	req.Header.Set("User-Agent", "BookNav-DeadlinkChecker/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		// fallback GET
		req, err = http.NewRequest(http.MethodGet, raw, nil)
		if err != nil {
			return false, 0, "request", err.Error(), int(time.Since(start).Milliseconds())
		}
		req.Header.Set("User-Agent", "BookNav-DeadlinkChecker/1.0")
		resp, err = s.httpClient.Do(req)
		if err != nil {
			return false, 0, "network", err.Error(), int(time.Since(start).Milliseconds())
		}
	}
	defer resp.Body.Close()
	ms = int(time.Since(start).Milliseconds())
	status = resp.StatusCode
	if status >= 200 && status < 400 {
		return true, status, "", "", ms
	}
	return false, status, "http", resp.Status, ms
}

func (s *JobService) discoverIcon(pageURL string) (string, error) {
	if err := assertPublicURL(pageURL); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "BookNav-IconFetcher/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fallbackFavicon(pageURL), nil
	}
	defer resp.Body.Close()
	body := io.LimitReader(resp.Body, 512*1024)
	doc, err := html.Parse(body)
	if err != nil {
		return fallbackFavicon(pageURL), nil
	}
	base, _ := url.Parse(pageURL)
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(a.Val)
				case "href":
					href = a.Val
				}
			}
			if href != "" && (strings.Contains(rel, "icon") || strings.Contains(rel, "shortcut")) {
				found = resolveURL(base, href)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if found != "" {
		return found, nil
	}
	return fallbackFavicon(pageURL), nil
}

func (s *JobService) downloadIcon(iconURL, dir string) (string, error) {
	if err := assertPublicURL(iconURL); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, iconURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "BookNav-IconFetcher/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	ext := ".ico"
	ct := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "png"):
		ext = ".png"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		ext = ".jpg"
	case strings.Contains(ct, "svg"):
		ext = ".svg"
	case strings.Contains(ct, "webp"):
		ext = ".webp"
	}
	name := hex.EncodeToString(sum[:8]) + ext
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (s *JobService) ListDeadlinks(ctx context.Context, batchID string, invalidOnly bool) ([]domain.DeadlinkCheck, error) {
	return s.deadlinks.ListByBatch(ctx, batchID, invalidOnly)
}

// FetchSiteInfo scrapes title/description/icon for quick-add.
func (s *JobService) FetchSiteInfo(raw string) (map[string]any, error) {
	raw = normalizeURL(raw)
	if err := assertPublicURL(raw); err != nil {
		return nil, apperr.New(apperr.Validation, err.Error())
	}
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 BookNav/1.0")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return map[string]any{"success": false, "url": raw, "message": err.Error()}, nil
	}
	defer resp.Body.Close()
	body := io.LimitReader(resp.Body, 1024*1024)
	doc, err := html.Parse(body)
	if err != nil {
		return map[string]any{"success": false, "url": raw, "message": err.Error()}, nil
	}
	title, desc := "", ""
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil && title == "" {
					title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				var name, prop, content string
				for _, a := range n.Attr {
					switch strings.ToLower(a.Key) {
					case "name":
						name = strings.ToLower(a.Val)
					case "property":
						prop = strings.ToLower(a.Val)
					case "content":
						content = a.Val
					}
				}
				if desc == "" && (name == "description" || prop == "og:description") {
					desc = strings.TrimSpace(content)
				}
				if title == "" && prop == "og:title" {
					title = strings.TrimSpace(content)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	icon, _ := s.discoverIcon(raw)
	return map[string]any{
		"success":     true,
		"url":         raw,
		"title":       title,
		"description": desc,
		"icon_url":    icon,
	}, nil
}

func fallbackFavicon(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/favicon.ico"
}

func resolveURL(base *url.URL, href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if base == nil {
		return href
	}
	return base.ResolveReference(u).String()
}

func assertPublicURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https allowed")
	}
	host := u.Hostname()
	if host == "localhost" || strings.HasSuffix(host, ".local") {
		return fmt.Errorf("private host blocked")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// allow unresolved at check time? block to be safe for SSRF
		return fmt.Errorf("host resolve failed")
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("private ip blocked")
		}
	}
	return nil
}
