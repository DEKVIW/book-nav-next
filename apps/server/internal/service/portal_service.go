package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/vector"
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

// SearchResult wraps websites with search meta.
type SearchResult struct {
	Websites []domain.Website `json:"websites"`
	Query    string           `json:"query"`
	Mode     string           `json:"mode"` // keyword | vector | hybrid
	AI       bool             `json:"ai"`
}

func (s *PortalService) Search(ctx context.Context, q string, user *domain.User, useAI bool) (*SearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return &SearchResult{Websites: []domain.Website{}, Query: q, Mode: "keyword"}, nil
	}
	if len(q) > 100 {
		q = q[:100]
	}

	// permission: AI / vector requires enable + auth policy
	aiOn, allowAnon := s.settings.AIEnabled(ctx)
	if useAI {
		if !aiOn {
			useAI = false
		} else if user == nil && !allowAnon {
			useAI = false
		}
	}

	// 1) keyword baseline
	keyword, err := s.websites.Search(ctx, q, 100)
	if err != nil {
		return nil, err
	}

	mode := "keyword"
	merged := keyword
	scoreMap := map[int64]float64{}
	usedVector := false
	usedIntent := false
	usedRerank := false

	// 2) vector + optional intent expand + optional LLM rerank
	if useAI && s.settings != nil {
		// 2a) vector recall
		if cfg, ready := s.settings.VectorConfig(ctx); ready {
			client := vector.NewClient(cfg)
			if emb, err := client.Embed(ctx, q); err == nil {
				if hits, err := client.Search(ctx, emb, cfg.MaxResults); err == nil && len(hits) > 0 {
					ids := make([]int64, 0, len(hits))
					for _, h := range hits {
						ids = append(ids, h.ID)
						scoreMap[h.ID] = h.Score
					}
					if vecSites, err := s.websites.GetByIDs(ctx, ids); err == nil {
						merged = mergeSearchResults(keyword, vecSites, scoreMap)
						usedVector = true
						if len(keyword) > 0 {
							mode = "hybrid"
						} else {
							mode = "vector"
						}
					}
				}
			}
			// embed/search failures silently fall back to keyword
		}

		// 2b) intent expand (LLM) — only for longer / natural-language queries
		providers, _ := s.settings.loadProviders(ctx)
		bindings := s.settings.loadTaskBindings(ctx)
		if shouldRunIntent(q) {
			if cand := s.settings.resolveTaskCandidate(providers, bindings, "intent"); cand != nil && cand.Model != "" {
				if intent, err := analyzeSearchIntent(ctx, cand, q); err == nil && intent != nil {
					extra := s.expandByIntent(ctx, intent)
					if len(extra) > 0 {
						merged = mergeSearchResults(merged, extra, scoreMap)
						usedIntent = true
						if mode == "keyword" {
							mode = "hybrid"
						}
					}
				} else if err != nil {
					slog.Debug("search intent skipped", "err", err)
				}
			}
		}

		// 2c) LLM rerank — only when enough candidates and rerank model configured
		if len(merged) >= 5 {
			if cand := s.settings.resolveTaskCandidate(providers, bindings, "rerank"); cand != nil && cand.Model != "" {
				limit := 40
				if len(merged) < limit {
					limit = len(merged)
				}
				items := make([]rerankItem, 0, limit)
				for i, w := range merged {
					if i >= limit {
						break
					}
					items = append(items, rerankItem{ID: w.ID, Title: w.Title, Desc: w.Description, URL: w.URL})
				}
				if ordered, err := rerankWithLLM(ctx, cand, q, items, 20); err == nil && len(ordered) > 0 {
					byID := map[int64]domain.Website{}
					for _, w := range merged {
						byID[w.ID] = w
					}
					var ranked []domain.Website
					seen := map[int64]bool{}
					for _, id := range ordered {
						if w, ok := byID[id]; ok && !seen[id] {
							seen[id] = true
							ranked = append(ranked, w)
						}
					}
					// append remainder in original order
					for _, w := range merged {
						if !seen[w.ID] {
							ranked = append(ranked, w)
						}
					}
					merged = ranked
					usedRerank = true
					if mode == "keyword" {
						mode = "hybrid"
					}
				} else if err != nil {
					slog.Debug("search rerank skipped", "err", err)
				}
			}
		}
	}

	var out []domain.Website
	for _, w := range merged {
		vids, _ := s.websites.ListViewers(ctx, w.ID)
		if domain.CanViewWebsite(&w, user, vids) {
			out = append(out, w)
		}
	}
	aiUsed := useAI && (usedVector || usedIntent || usedRerank)
	return &SearchResult{
		Websites: out,
		Query:    q,
		Mode:     mode,
		AI:       aiUsed,
	}, nil
}

// expandByIntent runs extra keyword searches from intent keywords / related terms.
func (s *PortalService) expandByIntent(ctx context.Context, intent *searchIntent) []domain.Website {
	var terms []string
	for _, k := range intent.Keywords {
		k = strings.TrimSpace(k)
		if k != "" {
			terms = append(terms, k)
		}
	}
	for _, k := range intent.RelatedTerms {
		k = strings.TrimSpace(k)
		if k != "" {
			terms = append(terms, k)
		}
	}
	if len(terms) > 8 {
		terms = terms[:8]
	}
	seen := map[int64]bool{}
	var out []domain.Website
	for _, t := range terms {
		if len([]rune(t)) < 2 {
			continue
		}
		list, err := s.websites.Search(ctx, t, 40)
		if err != nil {
			continue
		}
		for _, w := range list {
			if seen[w.ID] {
				continue
			}
			seen[w.ID] = true
			out = append(out, w)
		}
	}
	return out
}

// mergeSearchResults: vector hits first (by score), then keyword-only extras.
func mergeSearchResults(keyword, vectorHits []domain.Website, scores map[int64]float64) []domain.Website {
	seen := map[int64]bool{}
	var out []domain.Website

	// sort vector hits by score desc
	type scored struct {
		w     domain.Website
		score float64
	}
	var ranked []scored
	for _, w := range vectorHits {
		ranked = append(ranked, scored{w: w, score: scores[w.ID]})
	}
	// simple insertion sort (N small)
	for i := 1; i < len(ranked); i++ {
		j := i
		for j > 0 && ranked[j].score > ranked[j-1].score {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
			j--
		}
	}
	for _, r := range ranked {
		if seen[r.w.ID] {
			continue
		}
		seen[r.w.ID] = true
		out = append(out, r.w)
	}
	for _, w := range keyword {
		if seen[w.ID] {
			continue
		}
		seen[w.ID] = true
		out = append(out, w)
	}
	return out
}

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
