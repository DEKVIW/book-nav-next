package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"error", rec,
					"request_id", GetRequestID(r.Context()),
					"stack", string(debug.Stack()),
				)
				response.Internal(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
