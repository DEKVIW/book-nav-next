package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/repository"
)

type WebsiteService struct {
	websites   *repository.WebsiteRepo
	categories *repository.CategoryRepo
	oplog      *repository.OpLogRepo
	hooks      SiteSideEffects // optional: icon + vector lifecycle (legacy parity)
}

func NewWebsiteService(w *repository.WebsiteRepo, c *repository.CategoryRepo, o *repository.OpLogRepo) *WebsiteService {
	return &WebsiteService{websites: w, categories: c, oplog: o}
}

// SetSideEffects wires icon/vector hooks after construction (avoids init cycles).
func (s *WebsiteService) SetSideEffects(h SiteSideEffects) {
	s.hooks = h
}

type WebsiteInput struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	Icon        string  `json:"icon"`
	CategoryID  *int64  `json:"category_id"`
	IsFeatured  bool    `json:"is_featured"`
	IsPrivate   bool    `json:"is_private"`
	SortOrder   *int    `json:"sort_order"`
	ViewerIDs   []int64 `json:"viewer_ids"`
	Force       bool    `json:"force"`
}

func (s *WebsiteService) Create(ctx context.Context, user *domain.User, in WebsiteInput) (*domain.Website, error) {
	if user == nil || !user.Role.IsAdmin() {
		return nil, apperr.New(apperr.Forbidden, "需要管理员权限")
	}
	in.URL = normalizeURL(in.URL)
	in.Title = strings.TrimSpace(in.Title)
	if in.URL == "" {
		return nil, apperr.New(apperr.Validation, "URL 不能为空")
	}
	if in.Title == "" {
		in.Title = in.URL
	}
	if !in.Force {
		if exist, _ := s.websites.FindByURL(ctx, in.URL); exist != nil {
			return exist, apperr.New(apperr.Conflict, "链接已存在")
		}
	}
	order := 0
	if in.SortOrder != nil {
		order = *in.SortOrder
	} else {
		max, _ := s.websites.MaxSortOrder(ctx, in.CategoryID)
		order = max + 1
	}
	w := &domain.Website{
		Title:       in.Title,
		URL:         in.URL,
		Description: strings.TrimSpace(in.Description),
		Icon:        strings.TrimSpace(in.Icon),
		CategoryID:  in.CategoryID,
		CreatedBy:   user.ID,
		IsFeatured:  in.IsFeatured,
		IsPrivate:   in.IsPrivate,
		SortOrder:   order,
		IsValid:     true,
		ViewerIDs:   in.ViewerIDs,
	}
	if err := s.websites.Create(ctx, w); err != nil {
		return nil, err
	}
	// attach category name for logs / hooks
	if w.CategoryID != nil {
		if c, _ := s.categories.GetByID(ctx, *w.CategoryID); c != nil {
			w.CategoryName = c.Name
		}
	}
	_ = s.log(ctx, user.ID, "ADD", w, "{}")
	if s.hooks != nil {
		s.hooks.AfterWebsiteCreate(w.ID)
	}
	return w, nil
}

func (s *WebsiteService) Update(ctx context.Context, user *domain.User, id int64, in WebsiteInput) (*domain.Website, error) {
	w, err := s.websites.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, apperr.New(apperr.NotFound, "网站不存在")
	}
	if !domain.CanEditWebsite(w, user) {
		return nil, apperr.New(apperr.Forbidden, "无权编辑")
	}

	oldTitle := w.Title
	oldDesc := w.Description
	oldURL := w.URL
	oldIcon := w.Icon
	var oldCat int64
	if w.CategoryID != nil {
		oldCat = *w.CategoryID
	}

	if strings.TrimSpace(in.URL) != "" {
		w.URL = normalizeURL(in.URL)
	}
	if strings.TrimSpace(in.Title) != "" {
		w.Title = strings.TrimSpace(in.Title)
	}
	w.Description = strings.TrimSpace(in.Description)
	// Always apply provided fields from JSON layer (handler fills full input)
	w.Icon = strings.TrimSpace(in.Icon)
	w.CategoryID = in.CategoryID
	w.IsFeatured = in.IsFeatured
	w.IsPrivate = in.IsPrivate
	if in.SortOrder != nil {
		w.SortOrder = *in.SortOrder
	}
	if in.ViewerIDs != nil {
		w.ViewerIDs = in.ViewerIDs
	}
	if err := s.websites.Update(ctx, w); err != nil {
		return nil, err
	}
	if w.CategoryID != nil {
		if c, _ := s.categories.GetByID(ctx, *w.CategoryID); c != nil {
			w.CategoryName = c.Name
		}
	}
	details, _ := json.Marshal(in)
	_ = s.log(ctx, user.ID, "MODIFY", w, string(details))

	// legacy: title/description/category change → reindex vector
	var newCat int64
	if w.CategoryID != nil {
		newCat = *w.CategoryID
	}
	reindex := w.Title != oldTitle || w.Description != oldDesc || newCat != oldCat
	// legacy: icon or url change → resync icon (domain reuse / re-fetch)
	resyncIcon := w.URL != oldURL || w.Icon != oldIcon
	if s.hooks != nil && (reindex || resyncIcon) {
		s.hooks.AfterWebsiteUpdate(w.ID, reindex, resyncIcon)
	}
	return w, nil
}

func (s *WebsiteService) Delete(ctx context.Context, user *domain.User, id int64) error {
	w, err := s.websites.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if w == nil {
		return apperr.New(apperr.NotFound, "网站不存在")
	}
	if !domain.CanEditWebsite(w, user) {
		return apperr.New(apperr.Forbidden, "无权删除")
	}
	if err := s.websites.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.log(ctx, user.ID, "DELETE", w, "{}")
	if s.hooks != nil {
		s.hooks.AfterWebsiteDelete(id)
	}
	return nil
}

func (s *WebsiteService) BatchDelete(ctx context.Context, user *domain.User, ids []int64) (int64, error) {
	if user == nil || !user.Role.IsAdmin() {
		return 0, apperr.New(apperr.Forbidden, "需要管理员权限")
	}
	for _, id := range ids {
		if w, _ := s.websites.GetByID(ctx, id); w != nil {
			_ = s.log(ctx, user.ID, "DELETE", w, "{}")
		}
	}
	n, err := s.websites.DeleteMany(ctx, ids)
	if err != nil {
		return n, err
	}
	if s.hooks != nil && len(ids) > 0 {
		s.hooks.AfterWebsiteDelete(ids...)
	}
	return n, nil
}

func (s *WebsiteService) Reorder(ctx context.Context, user *domain.User, categoryID *int64, ids []int64) error {
	if user == nil || !user.Role.IsAdmin() {
		return apperr.New(apperr.Forbidden, "需要管理员权限")
	}
	return s.websites.UpdateOrders(ctx, categoryID, ids)
}

func (s *WebsiteService) ListAdmin(ctx context.Context, page, pageSize int, categoryID *int64, q string) ([]domain.Website, int, error) {
	return s.websites.ListAdmin(ctx, page, pageSize, categoryID, q)
}

func (s *WebsiteService) log(ctx context.Context, userID int64, action string, w *domain.Website, details string) error {
	uid := userID
	log := &domain.OperationLog{
		UserID:       &uid,
		Action:       action,
		WebsiteTitle: w.Title,
		WebsiteURL:   w.URL,
		WebsiteIcon:  w.Icon,
		CategoryID:   w.CategoryID,
		CategoryName: w.CategoryName,
		DetailsJSON:  details,
	}
	if action != "DELETE" {
		id := w.ID
		log.WebsiteID = &id
	}
	return s.oplog.Create(ctx, log)
}

func normalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	return u
}
