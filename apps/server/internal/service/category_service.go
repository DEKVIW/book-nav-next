package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/repository"
)

type CategoryService struct {
	repo     *repository.CategoryRepo
	websites *repository.WebsiteRepo
}

func NewCategoryService(repo *repository.CategoryRepo, websites *repository.WebsiteRepo) *CategoryService {
	return &CategoryService{repo: repo, websites: websites}
}

type CategoryInput struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	Color        string `json:"color"`
	SortOrder    *int   `json:"sort_order"`
	DisplayLimit *int   `json:"display_limit"`
	ParentID     *int64 `json:"parent_id"`
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	all, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	// attach website counts
	for i := range all {
		n, _ := s.websites.CountByCategory(ctx, all[i].ID)
		all[i].WebsiteCount = n
		all[i].DirectCount = n
	}
	// build tree for admin convenience
	byParent := map[int64][]domain.Category{}
	var roots []domain.Category
	for _, c := range all {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			byParent[*c.ParentID] = append(byParent[*c.ParentID], c)
		}
	}
	var attach func(domain.Category) domain.Category
	attach = func(c domain.Category) domain.Category {
		kids := byParent[c.ID]
		for i := range kids {
			kids[i] = attach(kids[i])
		}
		c.Children = kids
		return c
	}
	for i := range roots {
		roots[i] = attach(roots[i])
	}
	return roots, nil
}

func (s *CategoryService) Create(ctx context.Context, in CategoryInput) (*domain.Category, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apperr.New(apperr.Validation, "分类名称不能为空")
	}
	order := 0
	if in.SortOrder != nil {
		order = *in.SortOrder
	} else {
		max, _ := s.repo.MaxSortOrder(ctx, in.ParentID)
		order = max + 1
	}
	limit := 10
	if in.DisplayLimit != nil {
		limit = *in.DisplayLimit
	}
	icon := in.Icon
	if icon == "" {
		icon = "folder"
	}
	color := in.Color
	if color == "" {
		color = "#3DE7FF"
	}
	c := &domain.Category{
		Name:         name,
		Description:  in.Description,
		Icon:         icon,
		Color:        color,
		SortOrder:    order,
		DisplayLimit: limit,
		ParentID:     in.ParentID,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Update(ctx context.Context, id int64, in CategoryInput) (*domain.Category, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.New(apperr.NotFound, "分类不存在")
	}
	if strings.TrimSpace(in.Name) != "" {
		c.Name = strings.TrimSpace(in.Name)
	}
	c.Description = in.Description
	if in.Icon != "" {
		c.Icon = in.Icon
	}
	if in.Color != "" {
		c.Color = in.Color
	}
	if in.SortOrder != nil {
		c.SortOrder = *in.SortOrder
	}
	if in.DisplayLimit != nil {
		c.DisplayLimit = *in.DisplayLimit
	}
	// prevent self-parent
	if in.ParentID != nil && *in.ParentID == id {
		return nil, apperr.New(apperr.Validation, "不能将分类设为自己的父级")
	}
	c.ParentID = in.ParentID
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return apperr.New(apperr.NotFound, "分类不存在")
	}
	if n, _ := s.repo.CountChildren(ctx, id); n > 0 {
		return apperr.New(apperr.Validation, "请先删除或移动子分类")
	}
	if n, _ := s.websites.CountByCategory(ctx, id); n > 0 {
		return apperr.New(apperr.Validation, "分类下仍有链接，无法删除")
	}
	return s.repo.Delete(ctx, id)
}

// Reorder rewrites sort_order for a sibling group (same parent_id, or all roots).
// ids must be a permutation of one sibling set; first id gets highest sort_order (list DESC).
func (s *CategoryService) Reorder(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return apperr.New(apperr.Validation, "排序列表不能为空")
	}
	seen := map[int64]bool{}
	clean := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return apperr.New(apperr.Validation, "排序列表无效")
	}

	var parentKey string
	var parentID *int64
	for i, id := range clean {
		c, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if c == nil {
			return apperr.New(apperr.NotFound, "分类不存在")
		}
		key := parentGroupKey(c.ParentID)
		if i == 0 {
			parentKey = key
			parentID = c.ParentID
		} else if key != parentKey {
			return apperr.New(apperr.Validation, "只能对同一父级下的分类排序")
		}
	}

	all, err := s.repo.ListAll(ctx)
	if err != nil {
		return err
	}
	var siblings []int64
	for _, c := range all {
		same := (parentID == nil && c.ParentID == nil) ||
			(parentID != nil && c.ParentID != nil && *c.ParentID == *parentID)
		if same {
			siblings = append(siblings, c.ID)
		}
	}
	if len(siblings) != len(clean) {
		return apperr.New(apperr.Validation, "排序列表须包含该级全部同级分类")
	}
	sibSet := map[int64]bool{}
	for _, id := range siblings {
		sibSet[id] = true
	}
	for _, id := range clean {
		if !sibSet[id] {
			return apperr.New(apperr.Validation, "排序列表包含非同级分类")
		}
	}
	return s.repo.UpdateOrders(ctx, clean)
}

func parentGroupKey(parentID *int64) string {
	if parentID == nil {
		return ""
	}
	return strconv.FormatInt(*parentID, 10)
}
