package repository

import (
	"database/sql"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

func nullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullInt64(n sql.NullInt64) *int64 {
	if n.Valid {
		v := n.Int64
		return &v
	}
	return nil
}

func nullInt64Val(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0
}

func nullBool(n sql.NullInt64) bool {
	return n.Valid && n.Int64 != 0
}

func parseTime(s string) time.Time {
	t, err := clock.ParseTime(s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseTimePtr(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t := parseTime(ns.String)
	if t.IsZero() {
		return nil
	}
	return &t
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
