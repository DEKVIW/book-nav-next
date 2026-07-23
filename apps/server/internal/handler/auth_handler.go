package handler

import (
	"net/http"

	"github.com/booknav/book-nav/apps/server/internal/middleware"
	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
	"github.com/booknav/book-nav/apps/server/internal/service"
)

type AuthHandler struct {
	auth   *service.AuthService
	secure bool
}

func NewAuthHandler(auth *service.AuthService, secure bool) *AuthHandler {
	return &AuthHandler{auth: auth, secure: secure}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in service.LoginInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	user, sess, err := h.auth.Login(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	maxAge := 86400
	if in.Remember {
		maxAge = 30 * 86400
	}
	middleware.SetSessionCookie(w, sess.ID, maxAge, h.secure)
	response.OK(w, map[string]any{
		"user":       user.Public(),
		"csrf_token": sess.CSRFToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(middleware.SessionCookie)
	if c != nil {
		_ = h.auth.Logout(r.Context(), c.Value)
	}
	middleware.ClearSessionCookie(w)
	response.OKMessage(w, nil, "已退出")
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in service.RegisterInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	user, err := h.auth.Register(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, user.Public(), "注册成功，请登录")
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFrom(r.Context())
	sess := middleware.SessionFrom(r.Context())
	if u == nil {
		response.OK(w, map[string]any{"user": nil, "csrf_token": ""})
		return
	}
	token := ""
	if sess != nil {
		token = sess.CSRFToken
	}
	response.OK(w, map[string]any{"user": u.Public(), "csrf_token": token})
}

func (h *AuthHandler) CSRF(w http.ResponseWriter, r *http.Request) {
	sess := middleware.SessionFrom(r.Context())
	if sess == nil {
		response.OK(w, map[string]any{"csrf_token": ""})
		return
	}
	response.OK(w, map[string]any{"csrf_token": sess.CSRFToken})
}
