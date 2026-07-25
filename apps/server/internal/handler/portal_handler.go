package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/middleware"
	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
	"github.com/booknav/book-nav/apps/server/internal/service"
	"github.com/go-chi/chi/v5"
)

type PortalHandler struct {
	portal  *service.PortalService
	websites *service.WebsiteService
	categories *service.CategoryService
	jobs    *service.JobService
}

func NewPortalHandler(
	portal *service.PortalService,
	websites *service.WebsiteService,
	categories *service.CategoryService,
	jobs *service.JobService,
) *PortalHandler {
	return &PortalHandler{portal: portal, websites: websites, categories: categories, jobs: jobs}
}

func (h *PortalHandler) Home(w http.ResponseWriter, r *http.Request) {
	data, err := h.portal.Home(r.Context(), middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, data)
}

func (h *PortalHandler) CategoryWebsites(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	list, err := h.portal.CategoryWebsites(r.Context(), id, middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *PortalHandler) GetWebsite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	site, err := h.portal.GetWebsite(r.Context(), id, middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, site)
}

func (h *PortalHandler) Visit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	site, settings, err := h.portal.Visit(r.Context(), id, middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	user := middleware.UserFrom(r.Context())
	countdown := settings.TransitionTime
	if user != nil && user.Role.IsAdmin() {
		countdown = settings.AdminTransition
	}
	response.OK(w, map[string]any{
		"website":           site,
		"enable_transition": settings.EnableTransition,
		"countdown":         countdown,
		"remember_choice":   settings.TransitionRemember,
		"show_description":  settings.TransitionShowDesc,
		"theme":             settings.TransitionTheme,
		"color":             settings.TransitionColor,
		"ad1":               settings.TransitionAd1,
		"ad2":               settings.TransitionAd2,
	})
}

func (h *PortalHandler) Goto(w http.ResponseWriter, r *http.Request) {
	h.Visit(w, r)
}

func (h *PortalHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	useAI := r.URL.Query().Get("ai") == "1" || strings.EqualFold(r.URL.Query().Get("ai"), "true")
	res, err := h.portal.Search(r.Context(), q, middleware.UserFrom(r.Context()), useAI)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, res)
}

func (h *PortalHandler) SearchStream(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	useAI := r.URL.Query().Get("ai") == "1" || strings.EqualFold(r.URL.Query().Get("ai"), "true")
	// Prefer real flusher (may be wrapped by access-log middleware which must forward Flush).
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Last resort: same pipeline, one JSON response (EventSource cannot parse this).
		h.Search(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx: disable response buffering for SSE
	// Establish stream immediately so proxies/clients see event-stream, not a stalled request.
	_, _ = w.Write([]byte(": ok\n\n"))
	flusher.Flush()

	writeStage := func(res service.SearchResult) {
		payload, err := json.Marshal(res)
		if err != nil {
			return
		}
		_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
		flusher.Flush()
	}

	_, err := h.portal.SearchProgressive(r.Context(), q, middleware.UserFrom(r.Context()), useAI, func(stage service.SearchResult) {
		writeStage(stage)
	})
	if err != nil {
		payload, _ := json.Marshal(map[string]any{
			"stage":    "error",
			"error":    err.Error(),
			"websites": []any{},
			"query":    q,
		})
		_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
		flusher.Flush()
	}
}

func (h *PortalHandler) CheckURL(w http.ResponseWriter, r *http.Request) {
	site, exists, err := h.portal.CheckURL(r.Context(), r.URL.Query().Get("url"))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{"exists": exists, "website": site})
}

func (h *PortalHandler) FetchSite(w http.ResponseWriter, r *http.Request) {
	info, err := h.jobs.FetchSiteInfo(r.URL.Query().Get("url"))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, info)
}

func (h *PortalHandler) TranslateText(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text  string `json:"text"`
		Field string `json:"field"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	out, err := h.jobs.TranslateText(r.Context(), body.Text, body.Field)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, out)
}

func (h *PortalHandler) EnhanceSiteInfo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL         string `json:"url"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	out, err := h.jobs.EnhanceSiteInfo(r.Context(), body.URL, body.Title, body.Description)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, out)
}

func (h *PortalHandler) CreateWebsite(w http.ResponseWriter, r *http.Request) {
	var in service.WebsiteInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	site, err := h.websites.Create(r.Context(), middleware.UserFrom(r.Context()), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, site, "添加成功")
}

func (h *PortalHandler) UpdateWebsite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in service.WebsiteInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	site, err := h.websites.Update(r.Context(), middleware.UserFrom(r.Context()), id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, site, "更新成功")
}

func (h *PortalHandler) DeleteWebsite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.websites.Delete(r.Context(), middleware.UserFrom(r.Context()), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "已删除")
}

func (h *PortalHandler) ReorderWebsites(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CategoryID *int64  `json:"category_id"`
		IDs        []int64 `json:"ids"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	if err := h.websites.Reorder(r.Context(), middleware.UserFrom(r.Context()), body.CategoryID, body.IDs); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "排序已保存")
}

func (h *PortalHandler) ReorderCategories(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	if err := h.categories.Reorder(r.Context(), body.IDs); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "分类排序已保存")
}
