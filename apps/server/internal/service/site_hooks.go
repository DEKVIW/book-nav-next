package service

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/vector"
)

// SiteSideEffects is invoked after website CRUD (aligned with legacy Flask hooks).
type SiteSideEffects interface {
	AfterWebsiteCreate(siteID int64)
	AfterWebsiteUpdate(siteID int64, reindexVector, resyncIcon bool)
	AfterWebsiteDelete(siteIDs ...int64)
}

// Ensure JobService implements SiteSideEffects.
var _ SiteSideEffects = (*JobService)(nil)

func (s *JobService) AfterWebsiteCreate(siteID int64) {
	go s.runAfterCreate(siteID)
}

func (s *JobService) AfterWebsiteUpdate(siteID int64, reindexVector, resyncIcon bool) {
	go s.runAfterUpdate(siteID, reindexVector, resyncIcon)
}

func (s *JobService) AfterWebsiteDelete(siteIDs ...int64) {
	ids := append([]int64(nil), siteIDs...)
	go s.runAfterDelete(ids...)
}

func (s *JobService) runAfterCreate(siteID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := s.SyncWebsiteIcon(ctx, siteID, false); err != nil {
		slog.Debug("auto icon sync skipped/failed", "site_id", siteID, "err", err)
	}
	if err := s.IndexWebsite(ctx, siteID); err != nil {
		slog.Debug("auto vector index skipped/failed", "site_id", siteID, "err", err)
	}
}

func (s *JobService) runAfterUpdate(siteID int64, reindexVector, resyncIcon bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if resyncIcon {
		if err := s.SyncWebsiteIcon(ctx, siteID, true); err != nil {
			slog.Debug("icon resync skipped/failed", "site_id", siteID, "err", err)
		}
	}
	if reindexVector {
		if err := s.IndexWebsite(ctx, siteID); err != nil {
			slog.Debug("vector reindex skipped/failed", "site_id", siteID, "err", err)
		}
	}
}

func (s *JobService) runAfterDelete(siteIDs ...int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, id := range siteIDs {
		if err := s.DeleteWebsiteVector(ctx, id); err != nil {
			slog.Debug("vector delete skipped/failed", "site_id", id, "err", err)
		}
	}
}

// IndexWebsite embeds one website into Qdrant (no-op if vector not ready).
func (s *JobService) IndexWebsite(ctx context.Context, siteID int64) error {
	if s.settings == nil {
		return nil
	}
	cfg, ready := s.settings.VectorConfig(ctx)
	if !ready {
		return nil
	}
	site, err := s.websites.GetByID(ctx, siteID)
	if err != nil || site == nil {
		return err
	}
	client := vector.NewClient(cfg)
	text := buildWebsiteEmbedText(*site)
	emb, err := client.Embed(ctx, text)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"title":       site.Title,
		"url":         site.URL,
		"description": site.Description,
		"category":    site.CategoryName,
		"is_private":  site.IsPrivate,
	}
	return client.UpsertPoint(ctx, site.ID, emb, payload)
}

// DeleteWebsiteVector removes Qdrant point (best-effort).
// Runs whenever Qdrant URL is configured, even if vector search is disabled.
func (s *JobService) DeleteWebsiteVector(ctx context.Context, siteID int64) error {
	if s.settings == nil {
		return nil
	}
	cfg, _ := s.settings.VectorConfig(ctx)
	if cfg.QdrantURL == "" {
		return nil
	}
	client := vector.NewClient(cfg)
	return client.DeletePoint(ctx, siteID)
}

// SyncWebsiteIcon fetches / reuses domain icon for one site.
//
// force=false (create path):
//   - already /media/* → keep
//   - empty icon + auto_fetch → discover + optional local download
//   - remote http icon + sync_local → download and rewrite to /media
//
// force=true (url/icon changed):
//   - try domain reuse, then re-discover and save
func (s *JobService) SyncWebsiteIcon(ctx context.Context, siteID int64, force bool) error {
	site, err := s.websites.GetByID(ctx, siteID)
	if err != nil || site == nil {
		return err
	}

	autoFetch, syncLocal, providers := s.iconPrefs(ctx)
	icon := strings.TrimSpace(site.Icon)

	// Keep existing local media unless forced re-sync
	if !force && strings.HasPrefix(icon, "/media/") {
		return nil
	}

	// Domain reuse: same host shares one local path (legacy icon_asset domain_key behavior)
	if domain := domainKey(site.URL); domain != "" {
		if shared, err := s.websites.FindLocalIconByDomain(ctx, domain, site.ID); err == nil && shared != "" {
			if mediaFileExists(s.dataDir, shared) {
				if icon != shared {
					site.Icon = shared
					return s.websites.Update(ctx, site)
				}
				return nil
			}
		}
	}

	hasRemote := strings.HasPrefix(icon, "http://") || strings.HasPrefix(icon, "https://")

	// Remote icon provided, not forced: optionally localize
	if !force && hasRemote {
		if !syncLocal {
			return nil
		}
		return s.saveLocalIcon(site, icon)
	}

	// Non-http custom value (e.g. lucide name) — leave alone unless forced
	if icon != "" && !force && !hasRemote {
		return nil
	}

	// Empty icon without auto_fetch
	if icon == "" && !autoFetch && !force {
		return nil
	}

	// Discover from page / providers
	remote, err := s.discoverIconWithProviders(site.URL, providers)
	if err != nil || remote == "" {
		if hasRemote {
			remote = icon
		} else {
			return err
		}
	}

	if syncLocal {
		return s.saveLocalIcon(site, remote)
	}
	if site.Icon != remote {
		site.Icon = remote
		return s.websites.Update(ctx, site)
	}
	return nil
}

func (s *JobService) saveLocalIcon(site *domain.Website, remote string) error {
	iconDir := filepath.Join(s.dataDir, "uploads", "icons")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return err
	}
	local, err := s.downloadIcon(remote, iconDir)
	if err != nil {
		// last resort: keep remote URL
		if site.Icon != remote {
			site.Icon = remote
			_ = s.websites.Update(context.Background(), site)
		}
		return err
	}
	public := "/media/icons/" + filepath.Base(local)
	if site.Icon != public {
		site.Icon = public
		return s.websites.Update(context.Background(), site)
	}
	return nil
}

func (s *JobService) iconPrefs(ctx context.Context) (autoFetch, syncLocal bool, providers []IconSourceProvider) {
	autoFetch = true
	syncLocal = true
	providers = defaultIconSourceProviders()
	if s.settings == nil {
		return
	}
	adminMap, err := s.settings.GetNamespaceForAdmin(ctx, "icon")
	if err != nil {
		return
	}
	providers = mergeIconSourceProviders(adminMap["source_providers"])
	if v, ok := adminMap["auto_fetch"].(bool); ok {
		autoFetch = v
	}
	if v, ok := adminMap["sync_local"].(bool); ok {
		syncLocal = v
	}
	return
}

func domainKey(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Hostname() == "" {
		return domainKeyLoose(rawURL)
	}
	host := strings.ToLower(u.Hostname())
	return strings.TrimPrefix(host, "www.")
}

func domainKeyLoose(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if i := strings.Index(raw, "://"); i >= 0 {
		raw = raw[i+3:]
	}
	if i := strings.IndexAny(raw, "/?#"); i >= 0 {
		raw = raw[:i]
	}
	if i := strings.Index(raw, ":"); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimPrefix(raw, "www.")
}

func mediaFileExists(dataDir, mediaPath string) bool {
	mediaPath = strings.TrimSpace(mediaPath)
	if !strings.HasPrefix(mediaPath, "/media/") {
		return false
	}
	rel := strings.TrimPrefix(mediaPath, "/media/")
	abs := filepath.Join(dataDir, "uploads", filepath.FromSlash(rel))
	st, err := os.Stat(abs)
	return err == nil && !st.IsDir()
}
