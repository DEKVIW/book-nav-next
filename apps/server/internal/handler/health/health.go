package health

import (
	"net/http"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
	"github.com/booknav/book-nav/apps/server/internal/pkg/version"
)

// Handler 健康检查与版本信息。
type Handler struct {
	startedAt time.Time
}

func New() *Handler {
	return &Handler{startedAt: time.Now().UTC()}
}

// Live 存活探针：进程在即可。
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]any{
		"status": "ok",
	})
}

// Ready 就绪探针：后续挂 DB 检查。
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]any{
		"status": "ready",
	})
}

// Version 构建信息。
func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]any{
		"version":    version.Version,
		"commit":     version.Commit,
		"build_time": version.BuildTime,
		"uptime_sec": int(time.Since(h.startedAt).Seconds()),
	})
}
