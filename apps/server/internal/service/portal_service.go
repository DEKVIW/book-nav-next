package service

import (
	"context"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/repository"
)

type PortalService struct {
	categories *repository.CategoryRepo
	websites   *repository.WebsiteRepo
	settings   *SettingsService
	oplog      *repository.OpLogRepo
}

func NewPortalService(
	categories *repository.CategoryRepo,
	websites *repository.WebsiteRepo,
	settings *SettingsService,
	oplog *repository.OpLogRepo,
) *PortalService {
	return &PortalService{categories: categories, websites: websites, settings: settings, oplog: oplog}
}

func (s *PortalService) Home(ctx context.Context, user *domain.User) (*domain.HomeData, error) {
	allCats, err := s.categories.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	// index children
	childrenMap := map[int64][]domain.Category{}
	var roots []domain.Category
	for _, c := range allCats {
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			childrenMap[*c.ParentID] = append(childrenMap[*c.ParentID], c)
		}
	}

	// preload all websites once for filtering efficiency on small/medium datasets
	allSites, err := s.websites.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	viewersCache := map[int64][]int64{}
	for i := range allSites {
		vids, _ := s.websites.ListViewers(ctx, allSites[i].ID)
		viewersCache[allSites[i].ID] = vids
		allSites[i].ViewerIDs = vids
	}

	filter := func(list []domain.Website) []domain.Website {
		var out []domain.Website
		for _, w := range list {
			if domain.CanViewWebsite(&w, user, viewersCache[w.ID]) {
				out = append(out, w)
			}
		}
		return out
	}

	byCat := map[int64][]domain.Website{}
	for _, w := range filter(allSites) {
		if w.CategoryID == nil {
			continue
		}
		byCat[*w.CategoryID] = append(byCat[*w.CategoryID], w)
	}

	// sort already from ListAll? not by sort_order fully; re-sort per cat via repo would be better but OK for MVP
	// use ListByCategory for display order
	var result []domain.Category
	for _, root := range roots {
		children := childrenMap[root.ID]
		// attach counts and websites
		direct := byCat[root.ID]
		// ensure ordered
		if ordered, err := s.websites.ListByCategory(ctx, &root.ID, 10000); err == nil {
			direct = filter(ordered)
		}
		root.DirectCount = len(direct)
		childrenTotal := 0
		var childDTOs []domain.Category
		for _, ch := range children {
			chSites := byCat[ch.ID]
			if ordered, err := s.websites.ListByCategory(ctx, &ch.ID, 10000); err == nil {
				chSites = filter(ordered)
			}
			ch.WebsiteCount = len(chSites)
			ch.DirectCount = len(chSites)
			childrenTotal += ch.WebsiteCount
			childDTOs = append(childDTOs, ch)
		}
		root.Children = childDTOs
		root.TotalCountWithChildren = root.DirectCount + childrenTotal
		root.WebsiteCount = root.TotalCountWithChildren

		limit := root.DisplayLimit
		if limit <= 0 {
			limit = 10
		}

		if len(childDTOs) > 0 {
			if root.DirectCount < limit {
				// show first child websites
				first := childDTOs[0]
				sites := byCat[first.ID]
				if ordered, err := s.websites.ListByCategory(ctx, &first.ID, limit); err == nil {
					sites = filter(ordered)
				}
				if len(sites) > limit {
					sites = sites[:limit]
				}
				root.Websites = sites
				id := first.ID
				root.DisplayedSubcategoryID = &id
			} else {
				sites := direct
				if len(sites) > limit {
					sites = sites[:limit]
				}
				root.Websites = sites
				root.DisplayedSubcategoryID = nil
			}
		} else {
			sites := direct
			if len(sites) > limit {
				sites = sites[:limit]
			}
			root.Websites = sites
		}
		result = append(result, root)
	}

	featuredRaw, err := s.websites.ListFeatured(ctx, 6)
	if err != nil {
		return nil, err
	}
	featured := filter(featuredRaw)
	settings, err := s.settings.Public(ctx)
	if err != nil {
		return nil, err
	}
	var userPub map[string]any
	if user != nil {
		userPub = user.Public()
	}
	return &domain.HomeData{
		Categories: result,
		Featured:   featured,
		Settings:   settings,
		User:       userPub,
	}, nil
}

func (s *PortalService) CategoryWebsites(ctx context.Context, categoryID int64, user *domain.User) ([]domain.Website, error) {
	list, err := s.websites.ListAllByCategory(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	var out []domain.Website
	for _, w := range list {
		vids, _ := s.websites.ListViewers(ctx, w.ID)
		if domain.CanViewWebsite(&w, user, vids) {
			w.ViewerIDs = vids
			out = append(out, w)
		}
	}
	return out, nil
}

func (s *PortalService) GetWebsite(ctx context.Context, id int64, user *domain.User) (*domain.Website, error) {
	w, err := s.websites.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, apperr.New(apperr.NotFound, "网站不存在")
	}
	if !domain.CanViewWebsite(w, user, w.ViewerIDs) {
		return nil, apperr.New(apperr.Forbidden, "无权访问")
	}
	return w, nil
}

func (s *PortalService) Visit(ctx context.Context, id int64, user *domain.User) (*domain.Website, domain.PublicSettings, error) {
	w, err := s.GetWebsite(ctx, id, user)
	if err != nil {
		return nil, domain.PublicSettings{}, err
	}
	_ = s.websites.IncrementViews(ctx, id)
	settings, _ := s.settings.Public(ctx)
	return w, settings, nil
}

// Search / SearchProgressive / fusion live in ai_search_pipeline.go

func (s *PortalService) CheckURL(ctx context.Context, url string) (*domain.Website, bool, error) {
	url = normalizeURL(url)
	if url == "" {
		return nil, false, apperr.New(apperr.Validation, "URL 无效")
	}
	w, err := s.websites.FindByURL(ctx, url)
	if err != nil {
		return nil, false, err
	}
	if w == nil {
		return nil, false, nil
	}
	return w, true, nil
}
