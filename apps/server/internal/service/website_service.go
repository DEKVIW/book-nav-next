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
}

func NewWebsiteService(w *repository.WebsiteRepo, c *repository.CategoryRepo, o *repository.OpLogRepo) *WebsiteService {
	return &WebsiteService{websites: w, categories: c, oplog: o}
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
	_ = s.log(ctx, user.ID, "ADD", w, "{}")
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
	if strings.TrimSpace(in.URL) != "" {
		w.URL = normalizeURL(in.URL)
	}
	if strings.TrimSpace(in.Title) != "" {
		w.Title = strings.TrimSpace(in.Title)
	}
	w.Description = strings.TrimSpace(in.Description)
	if in.Icon != "" || in.Icon == "" {
		// allow clear only if key present - for simplicity always set if provided via pointer later
	}
	// Always apply provided fields from JSON layer (handler fills full input)
	if in.Title != "" {
		w.Title = strings.TrimSpace(in.Title)
	}
	w.Description = in.Description
	w.Icon = in.Icon
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
	details, _ := json.Marshal(in)
	_ = s.log(ctx, user.ID, "MODIFY", w, string(details))
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
	return s.websites.DeleteMany(ctx, ids)
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
