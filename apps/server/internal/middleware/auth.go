package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
	"github.com/booknav/book-nav/apps/server/internal/service"
)

const (
	SessionCookie = "booknav_session"
	ctxUserKey    ctxKey = "user"
	ctxSessionKey ctxKey = "session"
)

type AuthMiddleware struct {
	auth *service.AuthService
}

func NewAuthMiddleware(auth *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{auth: auth}
}

// LoadSession attaches user if cookie present (never blocks).
func (m *AuthMiddleware) LoadSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookie)
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		user, sess, err := m.auth.UserFromSession(r.Context(), c.Value)
		if err != nil || user == nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, user)
		ctx = context.WithValue(ctx, ctxSessionKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFrom(r.Context()) == nil {
			response.Unauthorized(w, "请先登录")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFrom(r.Context())
		if u == nil || !u.Role.IsAdmin() {
			response.Forbidden(w, "需要管理员权限")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) RequireSuper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFrom(r.Context())
		if u == nil || !u.Role.IsSuperAdmin() {
			response.Forbidden(w, "需要超级管理员权限")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CSRF protects state-changing methods when session exists.
func (m *AuthMiddleware) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		sess := SessionFrom(r.Context())
		if sess == nil {
			// public write endpoints (login/register) skip
			next.ServeHTTP(w, r)
			return
		}
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			token = r.FormValue("csrf_token")
		}
		if token == "" || token != sess.CSRFToken {
			response.Fail(w, http.StatusForbidden, "FORBIDDEN", "CSRF token invalid")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserFrom(ctx context.Context) *domain.User {
	if v, ok := ctx.Value(ctxUserKey).(*domain.User); ok {
		return v
	}
	return nil
}

func SessionFrom(ctx context.Context) *domain.Session {
	if v, ok := ctx.Value(ctxSessionKey).(*domain.Session); ok {
		return v
	}
	return nil
}

func SetSessionCookie(w http.ResponseWriter, sessionID string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   maxAge,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	// Clear both Secure and non-Secure variants (HTTP LAN vs HTTPS).
	for _, secure := range []bool{false, true} {
		http.SetCookie(w, &http.Cookie{
			Name:     SessionCookie,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
			MaxAge:   -1,
		})
	}
}

// RequestLooksHTTPS reports TLS or reverse-proxy HTTPS.
func RequestLooksHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if proto == "https" {
		return true
	}
	// Some proxies send comma-separated list
	if i := strings.IndexByte(proto, ','); i >= 0 {
		proto = strings.TrimSpace(proto[:i])
		if proto == "https" {
			return true
		}
	}
	return false
}

// CookieSecureForRequest decides Secure flag.
// mode: "true" always, "false" never, "auto" only when request is HTTPS.
func CookieSecureForRequest(r *http.Request, mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return RequestLooksHTTPS(r)
	}
}
