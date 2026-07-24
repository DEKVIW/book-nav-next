package handler

import (
	"database/sql"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/config"
	"github.com/booknav/book-nav/apps/server/internal/handler/health"
	"github.com/booknav/book-nav/apps/server/internal/middleware"
	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
	"github.com/booknav/book-nav/apps/server/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Deps struct {
	Config     *config.Config
	Auth       *service.AuthService
	Portal     *service.PortalService
	Websites   *service.WebsiteService
	Categories *service.CategoryService
	Admin      *service.AdminService
	Settings   *service.SettingsService
	Jobs       *service.JobService
	DB         *sql.DB
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	cfg := d.Config

	r.Use(middleware.Recover)
	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	if cfg.EnableAccessLog {
		r.Use(middleware.AccessLog)
	}

	origins := cfg.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authMW := middleware.NewAuthMiddleware(d.Auth)
	r.Use(authMW.LoadSession)

	healthH := health.New()
	// ready checks DB
	r.Get("/healthz", healthH.Live)
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if d.DB != nil {
			if err := d.DB.Ping(); err != nil {
				response.Fail(w, http.StatusServiceUnavailable, "INTERNAL", "db not ready")
				return
			}
		}
		healthH.Ready(w, r)
	})
	r.Get("/version", healthH.Version)

	// uploaded media
	mediaDir := filepath.Join(cfg.DataDir, "uploads")
	_ = os.MkdirAll(filepath.Join(mediaDir, "icons"), 0o755)
	_ = os.MkdirAll(filepath.Join(mediaDir, "avatars"), 0o755)
	r.Handle("/media/*", http.StripPrefix("/media/", http.FileServer(http.Dir(mediaDir))))

	authH := NewAuthHandler(d.Auth, cfg.CookieSecureMode())
	portalH := NewPortalHandler(d.Portal, d.Websites, d.Categories, d.Jobs)
	adminH := NewAdminHandler(d.Admin, d.Websites, d.Categories, d.Settings, d.Jobs)

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(authMW.CSRF)

		api.Get("/health", healthH.Live)
		api.Get("/version", healthH.Version)
		api.Get("/", func(w http.ResponseWriter, r *http.Request) {
			response.OK(w, map[string]any{"name": "BookNav API", "version": "v1"})
		})

		// auth
		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/login", authH.Login)
			ar.Post("/register", authH.Register)
			ar.Post("/logout", authH.Logout)
			ar.Get("/me", authH.Me)
			ar.Get("/csrf", authH.CSRF)
		})

		// portal
		api.Route("/portal", func(pr chi.Router) {
			pr.Get("/home", portalH.Home)
			pr.Get("/categories/{id}/websites", portalH.CategoryWebsites)
			pr.Get("/websites/{id}", portalH.GetWebsite)
			pr.Post("/websites/{id}/visit", portalH.Visit)
			pr.Get("/goto/{id}", portalH.Goto)
			pr.Get("/search", portalH.Search)
			pr.Get("/search/stream", portalH.SearchStream)
			pr.Get("/utils/check-url", portalH.CheckURL)
			pr.Get("/utils/fetch-site", portalH.FetchSite)

			pr.Group(func(wr chi.Router) {
				wr.Use(authMW.RequireAuth)
				wr.Use(authMW.RequireAdmin)
				wr.Post("/websites", portalH.CreateWebsite)
				wr.Patch("/websites/{id}", portalH.UpdateWebsite)
				wr.Delete("/websites/{id}", portalH.DeleteWebsite)
				wr.Put("/websites/order", portalH.ReorderWebsites)
				wr.Put("/categories/order", portalH.ReorderCategories)
			})
		})

		// admin
		api.Route("/admin", func(ar chi.Router) {
			ar.Use(authMW.RequireAuth)
			ar.Use(authMW.RequireAdmin)

			ar.Get("/stats", adminH.Stats)
			ar.Get("/websites", adminH.ListWebsites)
			ar.Post("/websites", adminH.CreateWebsite)
			ar.Patch("/websites/{id}", adminH.UpdateWebsite)
			ar.Delete("/websites/{id}", adminH.DeleteWebsite)
			ar.Post("/websites/batch-delete", adminH.BatchDeleteWebsites)

			ar.Get("/categories", adminH.ListCategories)
			ar.Post("/categories", adminH.CreateCategory)
			ar.Patch("/categories/{id}", adminH.UpdateCategory)
			ar.Delete("/categories/{id}", adminH.DeleteCategory)

			ar.Get("/invitations", adminH.ListInvites)
			ar.Post("/invitations", adminH.CreateInvites)
			ar.Delete("/invitations/{id}", adminH.DeleteInvite)

			ar.Get("/operation-logs", adminH.ListLogs)
			ar.Delete("/operation-logs/{id}", adminH.DeleteLog)
			ar.Post("/operation-logs/clear", adminH.ClearLogs)
			ar.Get("/jobs", adminH.ListJobs)
			ar.Get("/jobs/{id}", adminH.GetJob)
			ar.Post("/jobs/{id}/cancel", adminH.CancelJob)
			ar.Delete("/jobs/{id}", adminH.DeleteJob)
			ar.Post("/jobs/clear", adminH.ClearJobs)

			ar.Group(func(sr chi.Router) {
				sr.Use(authMW.RequireSuper)
				sr.Get("/users", adminH.ListUsers)
				sr.Patch("/users/{id}", adminH.UpdateUser)
				sr.Post("/users/{id}/avatar", adminH.UploadUserAvatar)
				sr.Delete("/users/{id}", adminH.DeleteUser)
				sr.Get("/settings/{namespace}", adminH.GetSettings)
				sr.Put("/settings/{namespace}", adminH.PutSettings)
				sr.Get("/export", adminH.Export)
				sr.Get("/export/json", adminH.ExportJSON)
				sr.Get("/export/config", adminH.ExportConfig)
				sr.Post("/import", adminH.Import)
				sr.Post("/import/legacy-db3", adminH.ImportLegacyDB3)
				sr.Post("/import/db", adminH.ImportDBUpload)
				sr.Post("/import/config", adminH.ImportConfig)
				sr.Post("/import/config/upload", adminH.ImportConfigUpload)
				// Static backup paths first — avoid /backups/{name} swallowing "config"
				sr.Get("/backups", adminH.ListBackups)
				sr.Post("/backups", adminH.CreateBackup)
				sr.Post("/backups/create-config", adminH.CreateConfigBackup)
				sr.Get("/backups/{name}", adminH.DownloadBackup)
				sr.Delete("/backups/{name}", adminH.DeleteBackup)
				sr.Post("/backups/{name}/restore", adminH.RestoreBackup)
				sr.Get("/webdav", adminH.ListWebDAV)
				sr.Post("/webdav", adminH.SaveWebDAV)
				sr.Delete("/webdav/{id}", adminH.DeleteWebDAV)
				sr.Post("/webdav/{id}/test", adminH.TestWebDAV)
				sr.Post("/webdav/{id}/upload", adminH.UploadBackupWebDAV)
				sr.Post("/webdav/{id}/run-backup", adminH.RunWebDAVBackup)
				sr.Get("/webdav/{id}/remote", adminH.ListRemoteWebDAV)
				sr.Get("/data/cleanup-stats", adminH.CleanupStats)
				sr.Post("/data/clear-websites", adminH.ClearWebsites)
				sr.Post("/data/clear-categories", adminH.ClearCategories)
				sr.Post("/data/clear-navigation", adminH.ClearNavigation)
				sr.Post("/data/clear-vectors", adminH.ClearVectors)
				sr.Post("/data/clear-icon-files", adminH.ClearIconFiles)
				sr.Post("/data/clear-avatar-files", adminH.ClearAvatarFiles)
				sr.Post("/data/clear-deadlinks", adminH.ClearDeadlinkRecords)
				sr.Post("/jobs/deadlink", adminH.StartDeadlink)
				sr.Post("/jobs/icons", adminH.StartIconJob)
				sr.Post("/jobs/vector-index", adminH.StartVectorJob)
				sr.Post("/vector/test", adminH.TestVector)
				sr.Get("/deadlinks", adminH.ListDeadlinks)

				// AI multi-provider management
				sr.Get("/ai/state", adminH.GetAIState)
				sr.Post("/ai/providers", adminH.SaveAIProvider)
				sr.Delete("/ai/providers/{id}", adminH.DeleteAIProvider)
				sr.Post("/ai/providers/{id}/detect", adminH.DetectAIProvider)
				sr.Post("/ai/providers/detect-all", adminH.DetectAllAIProviders)
				sr.Post("/ai/providers/{id}/test", adminH.TestAIProvider)
				sr.Put("/ai/task-bindings", adminH.SaveAITaskBindings)
				sr.Post("/ai/test-tasks", adminH.TestAITasks)
			})
		})
	})

	// SPA static
	if cfg.StaticDir != "" {
		mountSPA(r, cfg.StaticDir)
	} else if info, err := os.Stat("webdist"); err == nil && info.IsDir() {
		mountSPA(r, "webdist")
	}

	return r
}

func mountSPA(r chi.Router, staticDir string) {
	fileServer := http.FileServer(http.Dir(staticDir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		upath := path.Clean(req.URL.Path)
		if strings.HasPrefix(upath, "/api") || strings.HasPrefix(upath, "/media") {
			http.NotFound(w, req)
			return
		}
		if upath == "/" {
			http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
			return
		}
		full := filepath.Join(staticDir, filepath.FromSlash(strings.TrimPrefix(upath, "/")))
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, req)
			return
		}
		http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
	})
}
