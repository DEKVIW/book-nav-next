package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/middleware"
	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
	"github.com/booknav/book-nav/apps/server/internal/service"
	"github.com/go-chi/chi/v5"
)

type SearchShareHandler struct {
	shares *service.SearchShareService
}

func NewSearchShareHandler(shares *service.SearchShareService) *SearchShareHandler {
	return &SearchShareHandler{shares: shares}
}

type createSearchShareRequest struct {
	Title          string  `json:"title"`
	Query          string  `json:"query"`
	Mode           string  `json:"mode"`
	Summary        string  `json:"summary"`
	WebsiteIDs     []int64 `json:"website_ids"`
	ExpiresInDays  *int    `json:"expires_in_days"`
	IncludePrivate bool    `json:"include_private"`
}

func (h *SearchShareHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in createSearchShareRequest
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "请求参数无效")
		return
	}
	result, err := h.shares.Create(r.Context(), middleware.UserFrom(r.Context()), service.SearchShareCreateInput{
		Title:          in.Title,
		Query:          in.Query,
		Mode:           in.Mode,
		Summary:        in.Summary,
		WebsiteIDs:     in.WebsiteIDs,
		ExpiresInDays:  in.ExpiresInDays,
		IncludePrivate: in.IncludePrivate,
		Request:        r,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{
		"id":             result.Share.ID,
		"token":          result.Share.Token,
		"share_path":     "/share/" + result.Share.Token,
		"item_count":     result.Share.ItemCount,
		"excluded_count": result.ExcludedCount,
		"expires_at":     result.Share.ExpiresAt,
	})
}

func (h *SearchShareHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != domain.SearchShareStatusActive && status != domain.SearchShareStatusRevoked && status != domain.SearchShareStatusDeleted {
		response.BadRequest(w, "分享状态无效")
		return
	}
	items, total, err := h.shares.List(r.Context(), page, pageSize, status, r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *SearchShareHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	data, err := h.shares.GetPublic(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		writeErr(w, err)
		return
	}
	share := data.Share
	response.OK(w, map[string]any{
		"share": map[string]any{
			"title":      share.Title,
			"query":      share.Query,
			"mode":       share.Mode,
			"summary":    share.Summary,
			"status":     share.Status,
			"expires_at": share.ExpiresAt,
			"view_count": share.ViewCount,
			"item_count": len(data.Websites),
			"created_at": share.CreatedAt,
		},
		"websites": data.Websites,
	})
}

func (h *SearchShareHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := parseSearchShareID(r)
	if err != nil {
		response.BadRequest(w, "分享 ID 无效")
		return
	}
	if err := h.shares.Revoke(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "分享已撤销")
}

func (h *SearchShareHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseSearchShareID(r)
	if err != nil {
		response.BadRequest(w, "分享 ID 无效")
		return
	}
	if err := h.shares.Delete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "分享已删除")
}

func parseSearchShareID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}
