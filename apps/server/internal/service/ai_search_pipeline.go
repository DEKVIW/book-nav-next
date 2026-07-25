package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/vector"
)

// ---------------------------------------------------------------------------
// AI search pipeline (BookNav Next)
//
// True progressive SSE (not “wait for all then dump”):
//
//   1) keyword LIKE finishes first → emit stage=partial (cards appear ASAP)
//   2) vector + optional intent run in parallel → emit stage=partial again
//      when fused set grows / mode changes (append feel on the client)
//   3) optional LLM rerank → stage=final (reorder + summary only)
//
// Non-stream Search() still returns only the final frame.
// Intent keywords are NEVER the sole primary embedding input.
// ---------------------------------------------------------------------------

const (
	searchQueryMaxRunes       = 100
	searchKeywordLimit        = 100
	searchIntentTermMax       = 5
	searchIntentTermMinRunes  = 2
	searchIntentPerTermLimit  = 25
	searchIntentExtraCap      = 30
	searchRerankMinCandidates = 5
	searchRerankInputMax      = 40
	searchRerankOutputMax     = 20
	searchRRF_K               = 60
	searchExactBoost          = 0.35
	searchIntentHitBoost      = 0.08
	searchMultiQueryMax       = 2 // extra embeddings from intent keywords
	searchIntentTimeout       = 1200 * time.Millisecond
	searchRerankTimeout       = 1800 * time.Millisecond
	searchVectorTimeout       = 2500 * time.Millisecond
	searchVectorBatchSize     = 8 // emit fused list as vector hits accumulate (feel of “逐批出现”)
)

// SearchResult is the API payload for portal search.
type SearchResult struct {
	Websites   []domain.Website `json:"websites"`
	Query      string           `json:"query"`
	Mode       string           `json:"mode"` // keyword | vector | hybrid
	AI         bool             `json:"ai"`
	Stage      string           `json:"stage,omitempty"` // partial | final | error
	Summary    string           `json:"summary,omitempty"`
	Refined    bool             `json:"refined,omitempty"`
	UsedVector bool             `json:"used_vector,omitempty"`
	UsedIntent bool             `json:"used_intent,omitempty"`
	UsedRerank bool             `json:"used_rerank,omitempty"`
}

// SearchStageHandler receives progressive stages (SSE). May be nil.
type SearchStageHandler func(stage SearchResult)

// Search runs the full pipeline and returns the best available final result.
func (s *PortalService) Search(ctx context.Context, q string, user *domain.User, useAI bool) (*SearchResult, error) {
	return s.searchPipeline(ctx, q, user, useAI, nil)
}

// SearchProgressive runs initial then optional final, invoking onStage for each.
func (s *PortalService) SearchProgressive(ctx context.Context, q string, user *domain.User, useAI bool, onStage SearchStageHandler) (*SearchResult, error) {
	return s.searchPipeline(ctx, q, user, useAI, onStage)
}

func (s *PortalService) searchPipeline(ctx context.Context, q string, user *domain.User, useAI bool, onStage SearchStageHandler) (*SearchResult, error) {
	q = normalizeSearchQuery(q)
	if q == "" {
		empty := &SearchResult{Websites: []domain.Website{}, Query: q, Mode: "keyword", Stage: "final"}
		if onStage != nil {
			onStage(*empty)
		}
		return empty, nil
	}

	useAI = s.resolveUseAI(ctx, user, useAI)
	short := isShortSearchQuery(q)
	emit := onStage != nil

	emitPartial := func(list []domain.Website, mode string, ai, usedVec, usedIntent bool) SearchResult {
		frame := SearchResult{
			Websites:   s.filterVisible(ctx, user, list),
			Query:      q,
			Mode:       mode,
			AI:         ai,
			Stage:      "partial",
			UsedVector: usedVec,
			UsedIntent: usedIntent,
		}
		if emit {
			onStage(frame)
		}
		return frame
	}

	// --- 1) keyword first (fast path → first cards) ---
	kwList, err := s.websites.Search(ctx, q, searchKeywordLimit)
	if err != nil {
		return nil, err
	}
	if kwList == nil {
		kwList = []domain.Website{}
	}
	last := emitPartial(kwList, "keyword", false, false, false)

	// Non-AI: keyword is the whole answer.
	if !useAI || s.settings == nil {
		last.Stage = "final"
		if emit {
			onStage(last)
		}
		slog.Info("search done", "q_len", len([]rune(q)), "ai", false, "kw", len(kwList), "out", len(last.Websites), "mode", last.Mode)
		return &last, nil
	}

	// --- 2) vector + intent in parallel; push partial as each side lands ---
	type vecRes struct {
		sites  []domain.Website
		scores map[int64]float64
		ok     bool
	}
	type intentRes struct {
		intent *searchIntent
		extra  []domain.Website
		ok     bool
	}

	var (
		vecOut    vecRes
		intentOut intentRes
		wg        sync.WaitGroup
		mu        sync.Mutex
	)

	// live fusion state under mu
	vecSites := []domain.Website{}
	vecScores := map[int64]float64{}
	intentExtra := []domain.Website{}
	usedVec, usedIntent := false, false

	// Snapshot state and emit a partial frame (must not be called while holding mu).
	pushFused := func() {
		mu.Lock()
		vs := append([]domain.Website(nil), vecSites...)
		sc := make(map[int64]float64, len(vecScores))
		for id, v := range vecScores {
			sc[id] = v
		}
		ie := append([]domain.Website(nil), intentExtra...)
		uv, ui := usedVec, usedIntent
		mu.Unlock()

		fused := fuseSearchCandidates(q, kwList, vs, sc, ie)
		mode := searchMode(len(kwList) > 0, uv && len(vs) > 0, ui && len(ie) > 0)
		frame := SearchResult{
			Websites:   s.filterVisible(ctx, user, fused),
			Query:      q,
			Mode:       mode,
			AI:         uv || ui,
			Stage:      "partial",
			UsedVector: uv,
			UsedIntent: ui,
		}
		mu.Lock()
		last = frame
		mu.Unlock()
		if emit {
			onStage(frame)
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		vctx, cancel := context.WithTimeout(ctx, searchVectorTimeout)
		defer cancel()
		sites, scores, ok := s.recallVector(vctx, q)
		if !ok || len(sites) == 0 {
			return
		}
		sort.SliceStable(sites, func(i, j int) bool {
			return scores[sites[i].ID] > scores[sites[j].ID]
		})
		// Batch so the UI grows in steps (legacy-like progressive).
		for i := 0; i < len(sites); i += searchVectorBatchSize {
			end := i + searchVectorBatchSize
			if end > len(sites) {
				end = len(sites)
			}
			mu.Lock()
			vecSites = append([]domain.Website(nil), sites[:end]...)
			vecScores = scores
			usedVec = true
			mu.Unlock()
			pushFused()
			if vctx.Err() != nil {
				break
			}
		}
		vecOut = vecRes{sites: sites, scores: scores, ok: true}
	}()

	if !short && shouldRunIntent(q) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ictx, cancel := context.WithTimeout(ctx, searchIntentTimeout)
			defer cancel()
			intent, extra, ok := s.recallIntent(ictx, q)
			if !ok {
				return
			}
			mu.Lock()
			intentOut = intentRes{intent: intent, extra: extra, ok: true}
			intentExtra = extra
			usedIntent = true
			mu.Unlock()
			pushFused()
		}()
	}

	wg.Wait()

	// multi-query vector on intent terms (after both settled)
	mu.Lock()
	intentSnap := intentOut
	scoreSnap := make(map[int64]float64, len(vecScores))
	for id, v := range vecScores {
		scoreSnap[id] = v
	}
	mu.Unlock()
	if intentSnap.ok && intentSnap.intent != nil {
		extraVec, extraScores := s.recallVectorMultiQuery(ctx, intentSnap.intent, scoreSnap)
		if len(extraVec) > 0 {
			mu.Lock()
			vecSites = append(vecSites, extraVec...)
			if vecScores == nil {
				vecScores = map[int64]float64{}
			}
			for id, sc := range extraScores {
				if prev, ok := vecScores[id]; !ok || sc > prev {
					vecScores[id] = sc
				}
			}
			usedVec = true
			mu.Unlock()
			pushFused()
		}
	}

	mu.Lock()
	fused := fuseSearchCandidates(q, kwList, vecSites, vecScores, intentExtra)
	mode := searchMode(len(kwList) > 0, usedVec && len(vecSites) > 0, usedIntent && len(intentExtra) > 0)
	intentPtr := intentOut.intent
	uv, ui := usedVec, usedIntent
	vecN, intentN := len(vecSites), len(intentExtra)
	mu.Unlock()

	last = SearchResult{
		Websites:   s.filterVisible(ctx, user, fused),
		Query:      q,
		Mode:       mode,
		AI:         uv || ui,
		Stage:      "partial",
		UsedVector: uv,
		UsedIntent: ui,
	}

	// --- 3) final: gated rerank (reorder only) ---
	final := last
	final.Stage = "final"
	if !short && len(fused) >= searchRerankMinCandidates {
		rctx, cancel := context.WithTimeout(ctx, searchRerankTimeout)
		ordered, summary, ok := s.refineRerank(rctx, q, fused, intentPtr)
		cancel()
		if ok && len(ordered) > 0 {
			final.Websites = s.filterVisible(ctx, user, ordered)
			final.Summary = summary
			final.Refined = true
			final.UsedRerank = true
			final.AI = true
			if final.Mode == "keyword" {
				final.Mode = "hybrid"
			}
		}
	}

	if emit {
		onStage(final)
	}

	slog.Info("search done",
		"q_len", len([]rune(q)),
		"ai", useAI,
		"kw", len(kwList),
		"vec", vecN,
		"intent_extra", intentN,
		"fused", len(fused),
		"out", len(final.Websites),
		"mode", final.Mode,
		"refined", final.Refined,
	)
	_ = vecOut
	return &final, nil
}

func (s *PortalService) resolveUseAI(ctx context.Context, user *domain.User, useAI bool) bool {
	if !useAI || s.settings == nil {
		return false
	}
	aiOn, allowAnon := s.settings.AIEnabled(ctx)
	if !aiOn {
		return false
	}
	if user == nil && !allowAnon {
		return false
	}
	return true
}

func normalizeSearchQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	runes := []rune(q)
	if len(runes) > searchQueryMaxRunes {
		q = string(runes[:searchQueryMaxRunes])
	}
	return q
}

func isShortSearchQuery(q string) bool {
	n := 0
	for range q {
		n++
	}
	if n > 5 {
		return false
	}
	for _, w := range []string{"怎么", "如何", "哪里", "为什么", "什么", "哪个", "推荐", "有没有"} {
		if strings.Contains(q, w) {
			return false
		}
	}
	if strings.Contains(q, " ") || strings.Contains(q, "\t") {
		return false
	}
	return true
}

func searchMode(hasKW, hasVec, hasIntent bool) string {
	channels := 0
	if hasKW {
		channels++
	}
	if hasVec {
		channels++
	}
	if hasIntent {
		channels++
	}
	if channels >= 2 {
		return "hybrid"
	}
	if hasVec {
		return "vector"
	}
	return "keyword"
}

func (s *PortalService) recallVector(ctx context.Context, q string) ([]domain.Website, map[int64]float64, bool) {
	cfg, ready := s.settings.VectorConfig(ctx)
	if !ready {
		return nil, nil, false
	}
	client := vector.NewClient(cfg)
	emb, err := client.Embed(ctx, q)
	if err != nil {
		slog.Debug("search vector embed failed", "err", err)
		return nil, nil, false
	}
	hits, err := client.Search(ctx, emb, cfg.MaxResults)
	if err != nil || len(hits) == 0 {
		if err != nil {
			slog.Debug("search vector qdrant failed", "err", err)
		}
		return nil, nil, false
	}
	ids := make([]int64, 0, len(hits))
	scores := make(map[int64]float64, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ID)
		scores[h.ID] = h.Score
	}
	sites, err := s.websites.GetByIDs(ctx, ids)
	if err != nil {
		slog.Debug("search vector hydrate failed", "err", err)
		return nil, nil, false
	}
	return sites, scores, true
}

// recallVectorMultiQuery embeds up to N intent keywords and searches Qdrant (capped extras).
func (s *PortalService) recallVectorMultiQuery(ctx context.Context, intent *searchIntent, existing map[int64]float64) ([]domain.Website, map[int64]float64) {
	if intent == nil || s.settings == nil {
		return nil, nil
	}
	cfg, ready := s.settings.VectorConfig(ctx)
	if !ready {
		return nil, nil
	}
	terms := collectIntentTerms(intent, searchMultiQueryMax)
	if len(terms) == 0 {
		return nil, nil
	}
	client := vector.NewClient(cfg)
	scores := map[int64]float64{}
	idSet := map[int64]bool{}
	for _, term := range terms {
		if ctx.Err() != nil {
			break
		}
		emb, err := client.Embed(ctx, term)
		if err != nil {
			continue
		}
		// slightly stricter: use configured threshold as-is via client
		hits, err := client.Search(ctx, emb, minInt(20, cfg.MaxResults))
		if err != nil {
			continue
		}
		for _, h := range hits {
			if existing != nil {
				if _, ok := existing[h.ID]; ok {
					continue // already from primary query vector
				}
			}
			if prev, ok := scores[h.ID]; !ok || h.Score > prev {
				scores[h.ID] = h.Score
			}
			idSet[h.ID] = true
		}
	}
	if len(idSet) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sites, err := s.websites.GetByIDs(ctx, ids)
	if err != nil {
		return nil, nil
	}
	return sites, scores
}

func (s *PortalService) recallIntent(ctx context.Context, q string) (*searchIntent, []domain.Website, bool) {
	providers, _ := s.settings.loadProviders(ctx)
	bindings := s.settings.loadTaskBindings(ctx)
	cand := s.settings.resolveTaskCandidate(providers, bindings, "intent")
	if cand == nil || cand.Model == "" {
		return nil, nil, false
	}
	intent, err := analyzeSearchIntent(ctx, cand, q)
	if err != nil || intent == nil {
		if err != nil {
			slog.Debug("search intent failed", "err", err)
		}
		return nil, nil, false
	}
	extra := s.expandByIntentGated(ctx, intent)
	return intent, extra, true
}

func collectIntentTerms(intent *searchIntent, max int) []string {
	if intent == nil || max <= 0 {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	add := func(raw string) {
		t := strings.TrimSpace(raw)
		if t == "" || len([]rune(t)) < searchIntentTermMinRunes {
			return
		}
		if isNoiseSearchTerm(t) {
			return
		}
		key := strings.ToLower(t)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, t)
	}
	for _, k := range intent.Keywords {
		add(k)
		if len(out) >= max {
			return out
		}
	}
	for _, k := range intent.RelatedTerms {
		add(k)
		if len(out) >= max {
			return out
		}
	}
	return out
}

func (s *PortalService) expandByIntentGated(ctx context.Context, intent *searchIntent) []domain.Website {
	terms := collectIntentTerms(intent, searchIntentTermMax)
	if len(terms) == 0 {
		return nil
	}
	seen := map[int64]bool{}
	var out []domain.Website
	for _, t := range terms {
		if ctx.Err() != nil || len(out) >= searchIntentExtraCap {
			break
		}
		list, err := s.websites.Search(ctx, t, searchIntentPerTermLimit)
		if err != nil {
			continue
		}
		for _, w := range list {
			if seen[w.ID] {
				continue
			}
			seen[w.ID] = true
			out = append(out, w)
			if len(out) >= searchIntentExtraCap {
				break
			}
		}
	}
	return out
}

var searchStopwords = map[string]bool{
	"的": true, "了": true, "吗": true, "呢": true, "啊": true, "吧": true,
	"是": true, "在": true, "有": true, "和": true, "与": true, "或": true,
	"一个": true, "什么": true, "怎么": true, "如何": true, "哪些": true, "哪个": true,
	"the": true, "a": true, "an": true, "of": true, "and": true, "or": true, "to": true, "for": true,
}

func isNoiseSearchTerm(t string) bool {
	tl := strings.ToLower(strings.TrimSpace(t))
	if searchStopwords[tl] {
		return true
	}
	// pure punctuation
	onlyPunct := true
	for _, r := range t {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			onlyPunct = false
			break
		}
	}
	return onlyPunct
}

type scoredSite struct {
	w     domain.Website
	score float64
}

// fuseSearchCandidates: RRF over keyword / vector / intent lists + exact-match boosts.
func fuseSearchCandidates(
	q string,
	keyword, vectorSites []domain.Website,
	vectorScores map[int64]float64,
	intentExtra []domain.Website,
) []domain.Website {
	type acc struct {
		w     domain.Website
		rrf   float64
		vScore float64
		fromKW bool
		fromVec bool
		fromIntent bool
	}
	m := map[int64]*acc{}

	addRanked := func(list []domain.Website, weight float64, mark func(*acc)) {
		for i, w := range list {
			a, ok := m[w.ID]
			if !ok {
				a = &acc{w: w}
				m[w.ID] = a
			} else {
				// prefer richer row if needed
				a.w = w
			}
			a.rrf += weight * (1.0 / float64(searchRRF_K+i+1))
			if mark != nil {
				mark(a)
			}
		}
	}

	addRanked(keyword, 1.0, func(a *acc) { a.fromKW = true })
	// vector: sort by score desc first for stable RRF ranks
	if len(vectorSites) > 0 {
		vs := append([]domain.Website(nil), vectorSites...)
		sort.SliceStable(vs, func(i, j int) bool {
			return vectorScores[vs[i].ID] > vectorScores[vs[j].ID]
		})
		addRanked(vs, 1.0, func(a *acc) {
			a.fromVec = true
			a.vScore = vectorScores[a.w.ID]
		})
	}
	addRanked(intentExtra, 0.85, func(a *acc) { a.fromIntent = true })

	ql := strings.ToLower(strings.TrimSpace(q))
	out := make([]scoredSite, 0, len(m))
	for _, a := range m {
		score := a.rrf
		// exact / strong lexical boosts
		title := strings.ToLower(a.w.Title)
		url := strings.ToLower(a.w.URL)
		if ql != "" {
			if title == ql || strings.EqualFold(a.w.Title, q) {
				score += searchExactBoost
			} else if strings.HasPrefix(title, ql) || strings.Contains(title, ql) {
				score += searchExactBoost * 0.6
			}
			if strings.Contains(url, ql) {
				score += searchExactBoost * 0.25
			}
		}
		if a.fromIntent {
			score += searchIntentHitBoost
		}
		// light vector score blend (already in RRF; tiny extra for high sim)
		if a.fromVec && a.vScore >= 0.75 {
			score += 0.05
		}
		out = append(out, scoredSite{w: a.w, score: score})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score == out[j].score {
			return out[i].w.ID < out[j].w.ID
		}
		return out[i].score > out[j].score
	})
	websites := make([]domain.Website, 0, len(out))
	for _, s := range out {
		websites = append(websites, s.w)
	}
	return websites
}

func (s *PortalService) refineRerank(ctx context.Context, q string, candidates []domain.Website, intent *searchIntent) ([]domain.Website, string, bool) {
	if s.settings == nil || len(candidates) < searchRerankMinCandidates {
		return nil, "", false
	}
	providers, _ := s.settings.loadProviders(ctx)
	bindings := s.settings.loadTaskBindings(ctx)
	cand := s.settings.resolveTaskCandidate(providers, bindings, "rerank")
	if cand == nil || cand.Model == "" {
		return nil, "", false
	}

	limit := searchRerankInputMax
	if len(candidates) < limit {
		limit = len(candidates)
	}
	items := make([]rerankItem, 0, limit)
	known := make(map[int64]domain.Website, limit)
	for i := 0; i < limit; i++ {
		w := candidates[i]
		known[w.ID] = w
		items = append(items, rerankItem{ID: w.ID, Title: w.Title, Desc: w.Description, URL: w.URL})
	}

	intentLine := ""
	if intent != nil {
		intentLine = strings.TrimSpace(intent.Intent)
	}
	ordered, summary, err := rerankWithLLM(ctx, cand, q, intentLine, items, searchRerankOutputMax)
	if err != nil || len(ordered) == 0 {
		if err != nil {
			slog.Debug("search rerank skipped", "err", err)
		}
		return nil, "", false
	}

	// validate: only known ids, drop unknowns; require enough overlap
	var ranked []domain.Website
	seen := map[int64]bool{}
	for _, id := range ordered {
		w, ok := known[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		ranked = append(ranked, w)
	}
	if len(ranked) == 0 {
		return nil, "", false
	}
	// append remainder in original fused order (monotonic id set)
	for _, w := range candidates {
		if seen[w.ID] {
			continue
		}
		seen[w.ID] = true
		ranked = append(ranked, w)
	}
	return ranked, summary, true
}

func (s *PortalService) filterVisible(ctx context.Context, user *domain.User, list []domain.Website) []domain.Website {
	if list == nil {
		return []domain.Website{}
	}
	out := make([]domain.Website, 0, len(list))
	for _, w := range list {
		vids, _ := s.websites.ListViewers(ctx, w.ID)
		if domain.CanViewWebsite(&w, user, vids) {
			out = append(out, w)
		}
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
