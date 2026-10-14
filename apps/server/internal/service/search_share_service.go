package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
	"github.com/booknav/book-nav/apps/server/internal/repository"
)

const (
	searchShareMaxQueryRunes   = 100
	searchShareMaxTitleRunes   = 80
	searchShareMaxSummaryRunes = 2000
	searchShareMaxItems        = 40
	searchShareDefaultDays     = 7
	searchShareMaxDays         = 365
)

type SearchShareService struct {
	shares   *repository.SearchShareRepo
	websites *repository.WebsiteRepo
	settings *SettingsService
	guestMu  sync.Mutex
}

func NewSearchShareService(shares *repository.SearchShareRepo, websites *repository.WebsiteRepo, settings *SettingsService) *SearchShareService {
	return &SearchShareService{shares: shares, websites: websites, settings: settings}
}

type SearchShareCreateInput struct {
	Title          string
	Query          string
	Mode           string
	Summary        string
	WebsiteIDs     []int64
	ExpiresInDays  *int
	IncludePrivate bool
	Request        *http.Request
}

type SearchShareCreateResult struct {
	Share         *domain.SearchShare `json:"share"`
	ExcludedCount int                 `json:"excluded_count"`
}

type PublicSearchShare struct {
	Share    domain.SearchShare `json:"share"`
	Websites []domain.Website   `json:"websites"`
}

func (s *SearchShareService) Create(ctx context.Context, user *domain.User, in SearchShareCreateInput) (*SearchShareCreateResult, error) {
	policy := s.settings.SharePolicy(ctx)
	isAdmin := user != nil && user.Role.IsAdmin()
	if !isAdmin && !policy.GuestEnabled {
		return nil, apperr.New(apperr.Forbidden, "站长未开启游客分享")
	}
	if !isAdmin && in.IncludePrivate {
		return nil, apperr.New(apperr.Forbidden, "游客不能分享私人链接")
	}
	if !isAdmin && len(in.WebsiteIDs) > policy.GuestMaxItems {
		return nil, apperr.New(apperr.Validation, "游客分享结果不能超过配置上限")
	}
	ipHash := ""
	if !isAdmin {
		s.guestMu.Lock()
		defer s.guestMu.Unlock()
		ipHash = hashShareIP(clientIP(in.Request))
		if err := s.checkGuestLimits(ctx, ipHash, policy); err != nil {
			return nil, err
		}
	}

	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, apperr.New(apperr.Validation, "搜索词不能为空")
	}
	query = limitRunes(query, searchShareMaxQueryRunes)

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = query
	}
	title = limitRunes(title, searchShareMaxTitleRunes)

	summary := limitRunes(strings.TrimSpace(in.Summary), searchShareMaxSummaryRunes)
	mode := strings.TrimSpace(strings.ToLower(in.Mode))
	if mode != domain.SearchShareModeKeyword && mode != domain.SearchShareModeAI {
		return nil, apperr.New(apperr.Validation, "搜索模式无效")
	}
	if len(in.WebsiteIDs) == 0 {
		return nil, apperr.New(apperr.Validation, "至少需要一个搜索结果")
	}
	if len(in.WebsiteIDs) > searchShareMaxItems {
		return nil, apperr.New(apperr.Validation, "分享结果不能超过40条")
	}

	days := searchShareDefaultDays
	if in.ExpiresInDays != nil {
		days = *in.ExpiresInDays
	}
	if !isAdmin && (days < 1 || days > policy.GuestMaxDays) {
		return nil, apperr.New(apperr.Validation, "游客分享有效期最长为7天")
	}
	if isAdmin && (days < 0 || days > searchShareMaxDays) {
		return nil, apperr.New(apperr.Validation, "有效期必须为永久或在1到365天之间")
	}

	all, err := s.websites.GetByIDs(ctx, in.WebsiteIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]domain.Website, len(all))
	for _, site := range all {
		byID[site.ID] = site
	}

	publicIDs := make([]int64, 0, len(in.WebsiteIDs))
	seen := make(map[int64]struct{}, len(in.WebsiteIDs))
	excluded := 0
	for _, id := range in.WebsiteIDs {
		if _, ok := seen[id]; ok {
			excluded++
			continue
		}
		seen[id] = struct{}{}
		site, ok := byID[id]
		if !ok || (!in.IncludePrivate && site.IsPrivate) || !site.IsValid {
			excluded++
			continue
		}
		publicIDs = append(publicIDs, id)
	}
	if len(publicIDs) == 0 {
		return nil, apperr.New(apperr.Validation, "没有可公开分享的搜索结果")
	}
	if excluded > 0 {
		summary = ""
	}

	token, err := newSearchShareToken()
	if err != nil {
		return nil, err
	}
	var expiresAt *time.Time
	if days > 0 {
		expires := clock.NowUTC().AddDate(0, 0, days)
		expiresAt = &expires
	}

	share := &domain.SearchShare{
		Token:          token,
		CreatorType:    "guest",
		Title:          title,
		Query:          query,
		Mode:           mode,
		Summary:        summary,
		Status:         domain.SearchShareStatusActive,
		IncludePrivate: in.IncludePrivate,
		CreatedIPHash:  ipHash,
		ExpiresAt:      expiresAt,
	}
	if isAdmin {
		share.CreatorType = "admin"
		share.CreatedBy = &user.ID
	}
	if err := s.shares.Create(ctx, share, publicIDs); err != nil {
		return nil, err
	}

	return &SearchShareCreateResult{Share: share, ExcludedCount: excluded}, nil
}

func (s *SearchShareService) List(ctx context.Context, page, pageSize int, status, query string) ([]domain.SearchShare, int, error) {
	if _, err := s.shares.DeleteExpired(ctx, clock.NowUTC()); err != nil {
		return nil, 0, err
	}
	return s.shares.List(ctx, page, pageSize, status, query)
}

func (s *SearchShareService) CleanupExpired(ctx context.Context) error {
	_, err := s.shares.DeleteExpired(ctx, clock.NowUTC())
	return err
}

func (s *SearchShareService) RunCleanup(ctx context.Context) {
	const interval = time.Hour
	_ = s.CleanupExpired(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.CleanupExpired(ctx)
		}
	}
}

func (s *SearchShareService) GetPublic(ctx context.Context, token string) (*PublicSearchShare, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, apperr.New(apperr.NotFound, "分享不存在")
	}
	share, err := s.shares.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if share == nil || share.Status != domain.SearchShareStatusActive ||
		(share.ExpiresAt != nil && !clock.NowUTC().Before(*share.ExpiresAt)) {
		return nil, apperr.New(apperr.NotFound, "分享不存在或已失效")
	}

	items, err := s.shares.ListItems(ctx, share.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.WebsiteID)
	}
	all, err := s.websites.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]domain.Website, len(all))
	for _, site := range all {
		byID[site.ID] = site
	}
	websites := make([]domain.Website, 0, len(items))
	for _, item := range items {
		site, ok := byID[item.WebsiteID]
		if !ok || (site.IsPrivate && !share.IncludePrivate) || !site.IsValid {
			continue
		}
		websites = append(websites, site)
	}
	if len(websites) == 0 {
		return nil, apperr.New(apperr.NotFound, "分享结果已不可用")
	}
	if err := s.shares.IncrementViews(ctx, share.ID); err != nil {
		return nil, err
	}
	share.Token = ""
	share.ViewCount++
	return &PublicSearchShare{Share: *share, Websites: websites}, nil
}

func (s *SearchShareService) Revoke(ctx context.Context, id int64) error {
	if _, err := s.shares.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.shares.Revoke(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			return apperr.New(apperr.NotFound, "分享不存在或已撤销")
		}
		return err
	}
	return nil
}

func (s *SearchShareService) Delete(ctx context.Context, id int64) error {
	if err := s.shares.Delete(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			return apperr.New(apperr.NotFound, "分享不存在")
		}
		return err
	}
	return nil
}

func newSearchShareToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *SearchShareService) checkGuestLimits(ctx context.Context, ipHash string, policy SharePolicy) error {
	now := clock.NowUTC()
	hourly, err := s.shares.CountGuestCreatedSince(ctx, ipHash, now.Add(-time.Hour))
	if err != nil {
		return err
	}
	if hourly >= policy.GuestHourlyLimit {
		return apperr.New(apperr.RateLimited, "分享创建过于频繁，请稍后再试")
	}
	daily, err := s.shares.CountGuestCreatedSince(ctx, ipHash, now.Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if daily >= policy.GuestDailyLimit {
		return apperr.New(apperr.RateLimited, "今日分享创建次数已达上限")
	}
	active, err := s.shares.CountActiveGuestByIP(ctx, ipHash, now)
	if err != nil {
		return err
	}
	if active >= policy.GuestActiveLimit {
		return apperr.New(apperr.RateLimited, "当前有效分享数量已达上限")
	}
	return nil
}

func clientIP(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func hashShareIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}

func limitRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) > max {
		return string(runes[:max])
	}
	return value
}
