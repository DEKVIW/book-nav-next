package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Open opens SQLite with sensible pragmas. databaseURL like sqlite://./data/booknav.db
func Open(databaseURL, dataDir string) (*sql.DB, string, error) {
	path := sqlitePath(databaseURL, dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, "", fmt.Errorf("mkdir db dir: %w", err)
	}

	// modernc driver name is "sqlite"
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(30000)&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, "", err
	}
	sqlDB.SetMaxOpenConns(1) // SQLite writer safety for simple deploy
	sqlDB.SetMaxIdleConns(1)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, "", err
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=30000;",
	}
	for _, p := range pragmas {
		if _, err := sqlDB.Exec(p); err != nil {
			_ = sqlDB.Close()
			return nil, "", fmt.Errorf("pragma: %w", err)
		}
	}
	return sqlDB, path, nil
}

func sqlitePath(databaseURL, dataDir string) string {
	u := strings.TrimSpace(databaseURL)
	u = strings.TrimPrefix(u, "sqlite://")
	u = strings.TrimPrefix(u, "sqlite:///")
	if u == "" {
		return filepath.Join(dataDir, "booknav.db")
	}
	// handle /absolute and relative
	if strings.HasPrefix(u, "/") && len(u) > 2 && u[2] == ':' {
		// windows quirk from sqlite:////C:/... not used
	}
	return filepath.FromSlash(u)
}
