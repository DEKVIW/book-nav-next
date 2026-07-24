package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config 进程级配置，环境变量前缀 BOOKNAV_。
type Config struct {
	HTTPAddr        string
	DataDir         string
	DatabaseURL     string
	SecretKey       string
	AdminUsername   string
	AdminEmail      string
	AdminPassword   string
	CORSOrigins     []string
	Env             string // development | production
	StaticDir       string // 前端 dist；空则仅 API
	EnableAccessLog bool
	// CookieSecure: true/false force; empty = auto (HTTPS request or X-Forwarded-Proto only).
	// Never default Secure=true solely because ENV=production — breaks http://LAN access.
	CookieSecure string
}

// Load 从环境变量加载配置；可选读取 .env。
func Load() (*Config, error) {
	// 本地开发：依次尝试仓库根与当前目录的 .env
	_ = godotenv.Load()
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))
	_ = godotenv.Load(filepath.Join("..", "..", "..", ".env"))

	cfg := &Config{
		HTTPAddr:        getEnv("BOOKNAV_HTTP_ADDR", ":8080"),
		DataDir:         getEnv("BOOKNAV_DATA_DIR", "./data"),
		DatabaseURL:     getEnv("BOOKNAV_DATABASE_URL", ""),
		SecretKey:       getEnv("BOOKNAV_SECRET_KEY", "dev-only-change-me"),
		AdminUsername:   getEnv("BOOKNAV_ADMIN_USERNAME", "admin"),
		AdminEmail:      getEnv("BOOKNAV_ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:   getEnv("BOOKNAV_ADMIN_PASSWORD", "admin123"),
		Env:             getEnv("BOOKNAV_ENV", "development"),
		StaticDir:       getEnv("BOOKNAV_STATIC_DIR", ""),
		EnableAccessLog: getEnvBool("BOOKNAV_ACCESS_LOG", true),
		CookieSecure:    strings.TrimSpace(os.Getenv("BOOKNAV_COOKIE_SECURE")),
	}

	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "sqlite://" + filepath.ToSlash(filepath.Join(cfg.DataDir, "booknav.db"))
	}

	if raw := strings.TrimSpace(os.Getenv("BOOKNAV_CORS_ORIGINS")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if o := strings.TrimSpace(part); o != "" {
				cfg.CORSOrigins = append(cfg.CORSOrigins, o)
			}
		}
	} else if cfg.Env == "development" {
		cfg.CORSOrigins = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	return cfg, nil
}

func (c *Config) IsDev() bool {
	return c.Env == "development" || c.Env == "dev"
}

// CookieSecureMode returns "true" | "false" | "auto".
func (c *Config) CookieSecureMode() string {
	v := strings.ToLower(strings.TrimSpace(c.CookieSecure))
	switch v {
	case "1", "true", "yes", "on":
		return "true"
	case "0", "false", "no", "off":
		return "false"
	default:
		return "auto"
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
