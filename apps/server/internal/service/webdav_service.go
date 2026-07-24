package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
)

// WebDAVConfig multi-endpoint cloud backup (aligned with old backup_list).
type WebDAVConfig struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	URL                string `json:"webdav_url"`
	Username           string `json:"webdav_username"`
	Password           string `json:"webdav_password,omitempty"`
	PasswordConfigured bool   `json:"password_configured,omitempty"`
	Path               string `json:"webdav_path"`
	Enabled            bool   `json:"enabled"`
	AutoBackup         bool   `json:"auto_backup"`
	// BackupData / BackupConfig: which artifacts to create & upload (both may be true).
	BackupData      bool   `json:"backup_data"`
	BackupConfig    bool   `json:"backup_config"`
	BackupInterval  int    `json:"backup_interval"` // hours
	BackupKeepCount int    `json:"backup_keep_count"`
	LastBackupTime  string `json:"last_backup_time,omitempty"`
	LastBackupStatus string `json:"last_backup_status,omitempty"`
}

// normalizeWebDAVKinds defaults missing flags for older configs (data-only era → both on).
func normalizeWebDAVKinds(c *WebDAVConfig) {
	// If both false, treat as legacy unset → enable data (preserve old behavior) and config.
	if !c.BackupData && !c.BackupConfig {
		c.BackupData = true
		c.BackupConfig = true
	}
}

func (s *SettingsService) loadWebDAVConfigs(ctx context.Context) ([]WebDAVConfig, error) {
	raw, err := s.repo.Get(ctx, "webdav", "configs")
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []WebDAVConfig{}, nil
	}
	var list []WebDAVConfig
	if err := json.Unmarshal(raw, &list); err != nil {
		return []WebDAVConfig{}, nil
	}
	for i := range list {
		normalizeWebDAVKinds(&list[i])
	}
	return list, nil
}

// GetWebDAVRaw returns unmasked config (password included) for internal backup jobs.
func (s *SettingsService) GetWebDAVRaw(ctx context.Context, id int64) (*WebDAVConfig, error) {
	return s.getWebDAVRaw(ctx, id)
}

func (s *SettingsService) saveWebDAVConfigs(ctx context.Context, list []WebDAVConfig) error {
	return s.repo.Set(ctx, "webdav", "configs", list)
}

func maskWebDAV(c WebDAVConfig) WebDAVConfig {
	configured := strings.TrimSpace(c.Password) != ""
	c.PasswordConfigured = configured
	if configured {
		c.Password = secretSentinel
	} else {
		c.Password = ""
	}
	return c
}

func (s *SettingsService) ListWebDAV(ctx context.Context) ([]WebDAVConfig, error) {
	list, err := s.loadWebDAVConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]WebDAVConfig, 0, len(list))
	for _, c := range list {
		out = append(out, maskWebDAV(c))
	}
	return out, nil
}

func (s *SettingsService) SaveWebDAV(ctx context.Context, in WebDAVConfig) ([]WebDAVConfig, error) {
	list, err := s.loadWebDAVConfigs(ctx)
	if err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = "WebDAV"
	}
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if in.URL == "" {
		return nil, apperr.New(apperr.Validation, "请填写 WebDAV URL")
	}
	in.Path = strings.TrimSpace(in.Path)
	if in.Path == "" {
		in.Path = "/nav_backups/"
	}
	if !strings.HasPrefix(in.Path, "/") {
		in.Path = "/" + in.Path
	}
	if !strings.HasSuffix(in.Path, "/") {
		in.Path += "/"
	}
	if in.BackupInterval <= 0 {
		in.BackupInterval = 24
	}
	if in.BackupKeepCount <= 0 {
		in.BackupKeepCount = 10
	}
	if !in.BackupData && !in.BackupConfig {
		return nil, apperr.New(apperr.Validation, "请至少勾选「数据备份」或「配置备份」")
	}
	pw := strings.TrimSpace(in.Password)
	keep := pw == "" || pw == secretSentinel || strings.HasPrefix(pw, "****")

	if in.ID > 0 {
		found := false
		for i := range list {
			if list[i].ID != in.ID {
				continue
			}
			found = true
			old := list[i].Password
			list[i].Name = in.Name
			list[i].URL = in.URL
			list[i].Username = strings.TrimSpace(in.Username)
			list[i].Path = in.Path
			list[i].Enabled = in.Enabled
			list[i].AutoBackup = in.AutoBackup
			list[i].BackupData = in.BackupData
			list[i].BackupConfig = in.BackupConfig
			list[i].BackupInterval = in.BackupInterval
			list[i].BackupKeepCount = in.BackupKeepCount
			if keep {
				list[i].Password = old
			} else {
				list[i].Password = pw
			}
			break
		}
		if !found {
			return nil, apperr.New(apperr.NotFound, "配置不存在")
		}
	} else {
		var maxID int64
		for _, c := range list {
			if c.ID > maxID {
				maxID = c.ID
			}
		}
		in.ID = maxID + 1
		if keep {
			in.Password = ""
		} else {
			in.Password = pw
		}
		list = append(list, in)
	}
	if err := s.saveWebDAVConfigs(ctx, list); err != nil {
		return nil, err
	}
	return s.ListWebDAV(ctx)
}

func (s *SettingsService) DeleteWebDAV(ctx context.Context, id int64) ([]WebDAVConfig, error) {
	list, err := s.loadWebDAVConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := list[:0]
	found := false
	for _, c := range list {
		if c.ID == id {
			found = true
			continue
		}
		out = append(out, c)
	}
	if !found {
		return nil, apperr.New(apperr.NotFound, "配置不存在")
	}
	if err := s.saveWebDAVConfigs(ctx, out); err != nil {
		return nil, err
	}
	return s.ListWebDAV(ctx)
}

func (s *SettingsService) getWebDAVRaw(ctx context.Context, id int64) (*WebDAVConfig, error) {
	list, err := s.loadWebDAVConfigs(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i], nil
		}
	}
	return nil, apperr.New(apperr.NotFound, "配置不存在")
}

// TestWebDAV PROPFIND on remote path.
func (s *SettingsService) TestWebDAV(ctx context.Context, id int64) (map[string]any, error) {
	c, err := s.getWebDAVRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := webdavEnsureDir(ctx, *c); err != nil {
		return map[string]any{"ok": false, "message": "连接失败: " + err.Error()}, nil
	}
	return map[string]any{"ok": true, "message": "WebDAV 连接成功"}, nil
}

// UploadBackupToWebDAV uploads a local backup file to the given config.
func (s *SettingsService) UploadBackupToWebDAV(ctx context.Context, id int64, localPath, filename string) error {
	c, err := s.getWebDAVRaw(ctx, id)
	if err != nil {
		return err
	}
	if !c.Enabled {
		return apperr.New(apperr.Validation, "该云端配置已禁用")
	}
	if err := webdavEnsureDir(ctx, *c); err != nil {
		return apperr.New(apperr.Validation, "无法访问远端目录: "+err.Error())
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	remote := strings.TrimRight(c.URL, "/") + path.Join(c.Path, filename)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, remote, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/octet-stream")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		_ = s.markWebDAVStatus(ctx, id, false, err.Error())
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		_ = s.markWebDAVStatus(ctx, id, false, resp.Status)
		return fmt.Errorf("upload failed: %s %s", resp.Status, string(body))
	}
	_ = s.markWebDAVStatus(ctx, id, true, "上传成功 "+filename)
	return nil
}

func (s *SettingsService) markWebDAVStatus(ctx context.Context, id int64, ok bool, msg string) error {
	list, err := s.loadWebDAVConfigs(ctx)
	if err != nil {
		return err
	}
	status := "failed|" + msg
	if ok {
		status = "success|" + msg
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range list {
		if list[i].ID == id {
			list[i].LastBackupTime = now
			list[i].LastBackupStatus = status
			break
		}
	}
	return s.saveWebDAVConfigs(ctx, list)
}

func webdavEnsureDir(ctx context.Context, c WebDAVConfig) error {
	url := strings.TrimRight(c.URL, "/") + c.Path
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Depth", "0")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		mk, err := http.NewRequestWithContext(ctx, "MKCOL", url, nil)
		if err != nil {
			return err
		}
		mk.SetBasicAuth(c.Username, c.Password)
		r2, err := client.Do(mk)
		if err != nil {
			return err
		}
		defer r2.Body.Close()
		if r2.StatusCode >= 300 && r2.StatusCode != 405 {
			return fmt.Errorf("MKCOL %s", r2.Status)
		}
		return nil
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("认证失败 (%s)", resp.Status)
	}
	// 207 Multi-Status is success for PROPFIND
	if resp.StatusCode >= 300 && resp.StatusCode != 207 {
		return fmt.Errorf("PROPFIND %s", resp.Status)
	}
	return nil
}

// ListRemoteWebDAV lists files under path (simple PROPFIND).
func (s *SettingsService) ListRemoteWebDAV(ctx context.Context, id int64) ([]map[string]any, error) {
	c, err := s.getWebDAVRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(c.URL, "/") + c.Path
	payload := `<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:displayname/><d:getcontentlength/><d:getlastmodified/></d:prop></d:propfind>`
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 300 && resp.StatusCode != 207 {
		return nil, fmt.Errorf("list failed: %s", resp.Status)
	}
	text := string(body)
	var names []string
	for _, part := range strings.Split(text, "<") {
		low := strings.ToLower(part)
		if !strings.Contains(low, "href>") {
			continue
		}
		idx := strings.Index(part, ">")
		if idx < 0 {
			continue
		}
		rest := part[idx+1:]
		end := strings.Index(rest, "<")
		if end < 0 {
			continue
		}
		href := strings.TrimSpace(rest[:end])
		base := path.Base(href)
		if base == "" || base == "/" || base == "." {
			continue
		}
		lowBase := strings.ToLower(base)
		if strings.HasSuffix(lowBase, ".db") ||
			strings.HasSuffix(lowBase, ".db3") ||
			strings.HasSuffix(lowBase, ".zip") ||
			strings.HasSuffix(lowBase, ".json") ||
			strings.Contains(lowBase, "booknav") {
			names = append(names, base)
		}
	}
	seen := map[string]bool{}
	var out []map[string]any
	sort.Strings(names)
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, map[string]any{"name": n})
	}
	return out, nil
}
