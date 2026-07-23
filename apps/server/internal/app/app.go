package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/booknav/book-nav/apps/server/internal/config"
	"github.com/booknav/book-nav/apps/server/internal/db"
	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/handler"
	"github.com/booknav/book-nav/apps/server/internal/pkg/password"
	"github.com/booknav/book-nav/apps/server/internal/repository"
	"github.com/booknav/book-nav/apps/server/internal/service"
)

type App struct {
	Config *config.Config
	DB     *sql.DB
	DBPath string
	Router http.Handler
}

func New(cfg *config.Config) (*App, error) {
	sqlDB, dbPath, err := db.Open(cfg.DatabaseURL, cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	migDir := findMigrationsDir()
	if err := db.Migrate(sqlDB, migDir); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	// repos
	users := repository.NewUserRepo(sqlDB)
	sessions := repository.NewSessionRepo(sqlDB)
	invites := repository.NewInvitationRepo(sqlDB)
	categories := repository.NewCategoryRepo(sqlDB)
	websites := repository.NewWebsiteRepo(sqlDB)
	settingsRepo := repository.NewSettingsRepo(sqlDB)
	oplog := repository.NewOpLogRepo(sqlDB)
	jobs := repository.NewJobRepo(sqlDB)
	deadlinks := repository.NewDeadlinkRepo(sqlDB)

	// services
	authSvc := service.NewAuthService(users, sessions, invites)
	settingsSvc := service.NewSettingsService(settingsRepo)
	portalSvc := service.NewPortalService(categories, websites, settingsSvc, oplog)
	websiteSvc := service.NewWebsiteService(websites, categories, oplog)
	categorySvc := service.NewCategoryService(categories, websites)
	adminSvc := service.NewAdminService(users, invites, websites, categories, oplog, jobs, settingsSvc, cfg.DataDir, dbPath)
	jobSvc := service.NewJobService(jobs, websites, deadlinks, settingsSvc, cfg.DataDir)

	ctx := context.Background()
	if err := seed(ctx, cfg, users, settingsSvc, categories, websites); err != nil {
		slog.Warn("seed warning", "error", err)
	}

	router := handler.NewRouter(handler.Deps{
		Config:     cfg,
		Auth:       authSvc,
		Portal:     portalSvc,
		Websites:   websiteSvc,
		Categories: categorySvc,
		Admin:      adminSvc,
		Settings:   settingsSvc,
		Jobs:       jobSvc,
		DB:         sqlDB,
	})

	return &App{Config: cfg, DB: sqlDB, DBPath: dbPath, Router: router}, nil
}

func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

func seed(ctx context.Context, cfg *config.Config, users *repository.UserRepo, settings *service.SettingsService, cats *repository.CategoryRepo, sites *repository.WebsiteRepo) error {
	hash, err := password.Hash(cfg.AdminPassword)
	if err != nil {
		return err
	}
	if err := users.EnsureSuperAdmin(ctx, cfg.AdminUsername, cfg.AdminEmail, hash); err != nil {
		return err
	}
	if err := settings.EnsureDefaults(ctx); err != nil {
		return err
	}

	// demo content only when empty
	n, err := sites.Count(ctx)
	if err != nil || n > 0 {
		return err
	}
	allCats, _ := cats.ListAll(ctx)
	if len(allCats) > 0 {
		return nil
	}

	admin, err := users.GetByUsername(ctx, cfg.AdminUsername)
	if err != nil || admin == nil {
		return err
	}

	type seedCat struct {
		name, color string
		sites       []struct{ title, url, desc string }
	}
	demo := []seedCat{
		{name: "常用工具", color: "#3DE7FF", sites: []struct{ title, url, desc string }{
			{"GitHub", "https://github.com", "代码托管与协作"},
			{"Cloudflare", "https://dash.cloudflare.com", "边缘网络控制台"},
			{"ChatGPT", "https://chatgpt.com", "AI 助手"},
		}},
		{name: "开发资源", color: "#C084FC", sites: []struct{ title, url, desc string }{
			{"MDN", "https://developer.mozilla.org", "Web 文档"},
			{"Go Docs", "https://go.dev/doc/", "Go 语言文档"},
			{"Vue", "https://vuejs.org", "Vue.js 官网"},
		}},
		{name: "设计灵感", color: "#FFB020", sites: []struct{ title, url, desc string }{
			{"Dribbble", "https://dribbble.com", "设计作品"},
			{"Pinterest", "https://www.pinterest.com", "灵感收集"},
		}},
	}

	order := len(demo)
	for _, sc := range demo {
		c := &domain.Category{
			Name: sc.name, Color: sc.color, Icon: "folder",
			SortOrder: order, DisplayLimit: 10,
		}
		order--
		if err := cats.Create(ctx, c); err != nil {
			return err
		}
		so := len(sc.sites)
		for _, ss := range sc.sites {
			cid := c.ID
			w := &domain.Website{
				Title: ss.title, URL: ss.url, Description: ss.desc,
				CategoryID: &cid, CreatedBy: admin.ID,
				SortOrder: so, IsValid: true, IsFeatured: so == len(sc.sites),
			}
			so--
			if err := sites.Create(ctx, w); err != nil {
				return err
			}
		}
	}
	slog.Info("seeded demo categories and websites")
	return nil
}

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		filepath.Join("apps", "server", "migrations"),
		filepath.Join("..", "migrations"),
		filepath.Join("..", "..", "migrations"),
	}
	// also relative to executable
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "migrations"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "migrations"
}
