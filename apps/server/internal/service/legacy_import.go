package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/password"
	_ "modernc.org/sqlite"
)

// ImportLegacyDB3 imports an old Flask BookNav SQLite export (.db3).
// mode: "merge" | "replace"
func (s *AdminService) ImportLegacyDB3(ctx context.Context, path string, mode string, actor *domain.User) (map[string]int, error) {
	if actor == nil || !actor.Role.IsSuperAdmin() {
		return nil, apperr.New(apperr.Forbidden, "需要超级管理员")
	}
	if mode != "replace" && mode != "merge" {
		mode = "merge"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, apperr.New(apperr.NotFound, "导出文件不存在: "+abs)
	}

	src, err := sql.Open("sqlite", "file:"+filepath.ToSlash(abs)+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer src.Close()
	if err := src.Ping(); err != nil {
		return nil, fmt.Errorf("open legacy db: %w", err)
	}

	// detect format: need category + website tables
	var n int
	if err := src.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='website'`).Scan(&n); err != nil || n == 0 {
		return nil, apperr.New(apperr.Validation, "不是有效的 BookNav 导出（缺少 website 表）")
	}

	if mode == "replace" {
		if err := s.websites.ClearAll(ctx); err != nil {
			return nil, err
		}
		// clear categories
		cats, _ := s.categories.ListAll(ctx)
		// delete children first
		for _, c := range cats {
			if c.ParentID != nil {
				_ = s.categories.Delete(ctx, c.ID)
			}
		}
		cats, _ = s.categories.ListAll(ctx)
		for _, c := range cats {
			_ = s.categories.Delete(ctx, c.ID)
		}
	}

	// owner
	ownerID := actor.ID
	if u, err := s.importLegacyAdmin(ctx, src); err == nil && u != nil {
		ownerID = u.ID
	}

	// categories
	type legCat struct {
		ID, Parent sql.NullInt64
		Name, Desc, Icon, Color string
		Order, Limit int
	}
	rows, err := src.Query(`SELECT id, name, COALESCE(description,''), COALESCE(icon,''), COALESCE(color,'#3DE7FF'),
		COALESCE("order",0), COALESCE(display_limit,10), parent_id FROM category ORDER BY parent_id IS NOT NULL, "order" DESC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("read category: %w", err)
	}
	var legCats []legCat
	for rows.Next() {
		var c legCat
		if err := rows.Scan(&c.ID, &c.Name, &c.Desc, &c.Icon, &c.Color, &c.Order, &c.Limit, &c.Parent); err != nil {
			rows.Close()
			return nil, err
		}
		legCats = append(legCats, c)
	}
	rows.Close()

	idMap := map[int64]int64{}
	createdCats := 0
	catSvc := NewCategoryService(s.categories, s.websites)
	// first pass: create all as roots; second pass attach parents
	for _, c := range legCats {
		if !c.ID.Valid {
			continue
		}
		oldID := c.ID.Int64
		order := c.Order
		limit := c.Limit
		in := CategoryInput{
			Name: c.Name, Description: c.Desc, Icon: c.Icon, Color: c.Color,
			SortOrder: &order, DisplayLimit: &limit,
		}
		created, err := catSvc.Create(ctx, in)
		if err != nil {
			continue
		}
		idMap[oldID] = created.ID
		createdCats++
	}
	// fix parents
	for _, c := range legCats {
		if !c.ID.Valid || !c.Parent.Valid {
			continue
		}
		newID, ok := idMap[c.ID.Int64]
		if !ok {
			continue
		}
		newParent, ok := idMap[c.Parent.Int64]
		if !ok {
			continue
		}
		cat, _ := s.categories.GetByID(ctx, newID)
		if cat == nil {
			continue
		}
		cat.ParentID = &newParent
		_ = s.categories.Update(ctx, cat)
	}

	// websites
	wrows, err := src.Query(`SELECT id, title, url, COALESCE(description,''), COALESCE(icon,''),
		COALESCE(views,0), COALESCE(is_featured,0), COALESCE(sort_order,0), category_id,
		COALESCE(created_by_id,0), COALESCE(is_private,0), COALESCE(visible_to,''),
		COALESCE(is_valid,1) FROM website`)
	if err != nil {
		return nil, fmt.Errorf("read website: %w", err)
	}
	defer wrows.Close()

	createdSites := 0
	skipped := 0
	for wrows.Next() {
		var (
			id, views, featured, sortOrder, createdBy, private, valid int64
			title, url, desc, icon, visibleTo string
			catID sql.NullInt64
		)
		if err := wrows.Scan(&id, &title, &url, &desc, &icon, &views, &featured, &sortOrder, &catID,
			&createdBy, &private, &visibleTo, &valid); err != nil {
			return nil, err
		}
		url = strings.TrimSpace(url)
		if url == "" {
			skipped++
			continue
		}
		if mode == "merge" {
			if exist, _ := s.websites.FindByURL(ctx, normalizeURL(url)); exist != nil {
				skipped++
				continue
			}
		}
		var newCat *int64
		if catID.Valid {
			if nid, ok := idMap[catID.Int64]; ok {
				newCat = &nid
			}
		}
		so := int(sortOrder)
		w := &domain.Website{
			Title:       strings.TrimSpace(title),
			URL:         normalizeURL(url),
			Description: desc,
			Icon:        icon,
			CategoryID:  newCat,
			CreatedBy:   ownerID,
			IsFeatured:  featured != 0,
			IsPrivate:   private != 0,
			SortOrder:   so,
			Views:       views,
			IsValid:     valid != 0,
			ViewerIDs:   parseVisibleTo(visibleTo),
		}
		if w.Title == "" {
			w.Title = w.URL
		}
		if err := s.websites.Create(ctx, w); err != nil {
			skipped++
			continue
		}
		createdSites++
	}

	// settings (best effort)
	_ = s.importLegacySettings(ctx, src)

	return map[string]int{
		"categories": createdCats,
		"websites":   createdSites,
		"skipped":    skipped,
	}, nil
}

func (s *AdminService) importLegacyAdmin(ctx context.Context, src *sql.DB) (*domain.User, error) {
	// pick first superadmin-like user
	row := src.QueryRow(`SELECT username, email, password_hash,
		COALESCE(is_admin,0), COALESCE(is_superadmin,0), COALESCE(avatar,'')
		FROM user ORDER BY is_superadmin DESC, is_admin DESC, id ASC LIMIT 1`)
	var username, email, hash, avatar string
	var isAdmin, isSuper int
	if err := row.Scan(&username, &email, &hash, &isAdmin, &isSuper, &avatar); err != nil {
		return s.users.GetByID(ctx, 1)
	}
	existing, _ := s.users.GetByUsername(ctx, username)
	if existing != nil {
		return existing, nil
	}
	// keep old hash if bcrypt-compatible; else rehash default
	if !strings.HasPrefix(hash, "$2") {
		h, err := password.Hash("admin123")
		if err != nil {
			return nil, err
		}
		hash = h
	}
	role := domain.RoleUser
	if isSuper != 0 {
		role = domain.RoleSuperAdmin
	} else if isAdmin != 0 {
		role = domain.RoleAdmin
	}
	u := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Avatar:       avatar,
		Role:         role,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return s.users.GetByID(ctx, 1)
	}
	return u, nil
}

func (s *AdminService) importLegacySettings(ctx context.Context, src *sql.DB) error {
	row := src.QueryRow(`SELECT COALESCE(site_name,'BookNav'), COALESCE(site_subtitle,''),
		COALESCE(site_logo,''), COALESCE(site_favicon,''), COALESCE(footer_content,''),
		COALESCE(ai_search_enabled,0), COALESCE(ai_search_allow_anonymous,0),
		COALESCE(enable_transition,0), COALESCE(transition_time,5), COALESCE(admin_transition_time,0),
		COALESCE(announcement_enabled,0), COALESCE(announcement_title,''), COALESCE(announcement_content,'')
		FROM site_settings LIMIT 1`)
	var name, subtitle, logo, favicon, footer, annTitle, annContent string
	var aiOn, aiAnon, trOn, trTime, trAdmin, annOn int
	if err := row.Scan(&name, &subtitle, &logo, &favicon, &footer, &aiOn, &aiAnon, &trOn, &trTime, &trAdmin, &annOn, &annTitle, &annContent); err != nil {
		return nil
	}
	_ = s.settings.SetNamespace(ctx, "site", map[string]any{
		"name": name, "subtitle": subtitle, "logo": logo, "favicon": favicon, "footer": footer,
	})
	_ = s.settings.SetNamespace(ctx, "ai", map[string]any{
		"enabled": aiOn != 0, "allow_anonymous": aiAnon != 0,
	})
	_ = s.settings.SetNamespace(ctx, "transition", map[string]any{
		"enable": trOn != 0, "time": trTime, "admin_time": trAdmin,
	})
	_ = s.settings.SetNamespace(ctx, "announcement", map[string]any{
		"enabled": annOn != 0, "title": annTitle, "content": annContent,
	})
	return nil
}

func parseVisibleTo(s string) []int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var ids []int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(p, "%d", &id); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// ImportLegacyFromDataDir looks for booknav_export_*.db3 under dataDir or given name.
func (s *AdminService) ResolveImportPath(dataDir, filename string) (string, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return "", apperr.New(apperr.Validation, "请指定导出文件名")
	}
	// allow only under dataDir
	candidates := []string{
		filepath.Join(dataDir, filename),
		filepath.Join(dataDir, "imports", filename),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", apperr.New(apperr.NotFound, "在 data 目录未找到文件: "+filename)
}

