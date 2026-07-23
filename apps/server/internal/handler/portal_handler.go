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
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.Search(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	res, err := h.portal.Search(r.Context(), q, middleware.UserFrom(r.Context()), useAI)
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"stage": "error", "error": err.Error(), "websites": []any{}})
		_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
		flusher.Flush()
		return
	}
	list := res.Websites
	half := list
	if len(list) > 5 {
		half = list[:5]
	}
	for _, stage := range []struct {
		name string
		data any
	}{
		{"initial", half},
		{"enhanced", list},
		{"final", list},
	} {
		payload, _ := json.Marshal(map[string]any{
			"stage":    stage.name,
			"websites": stage.data,
			"meta": map[string]any{
				"query": res.Query,
				"ai":    res.AI,
				"mode":  res.Mode,
			},
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
