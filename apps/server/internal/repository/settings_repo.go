package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context, namespace, key string) (json.RawMessage, error) {
	var v string
	err := r.db.QueryRowContext(ctx, `SELECT value_json FROM settings_kv WHERE namespace=? AND key=?`, namespace, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(v), nil
}

func (r *SettingsRepo) Set(ctx context.Context, namespace, key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	now := clock.NowRFC3339()
	_, err = r.db.ExecContext(ctx, `
INSERT INTO settings_kv(namespace, key, value_json, updated_at) VALUES(?,?,?,?)
ON CONFLICT(namespace, key) DO UPDATE SET value_json=excluded.value_json, updated_at=excluded.updated_at`,
		namespace, key, string(b), now,
	)
	return err
}

func (r *SettingsRepo) GetNamespace(ctx context.Context, namespace string) (map[string]json.RawMessage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value_json FROM settings_kv WHERE namespace=?`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = json.RawMessage(v)
	}
	return out, rows.Err()
}

func (r *SettingsRepo) SetNamespace(ctx context.Context, namespace string, values map[string]any) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := clock.NowRFC3339()
	for k, v := range values {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO settings_kv(namespace, key, value_json, updated_at) VALUES(?,?,?,?)
ON CONFLICT(namespace, key) DO UPDATE SET value_json=excluded.value_json, updated_at=excluded.updated_at`,
			namespace, k, string(b), now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func DecodeSetting[T any](raw json.RawMessage, def T) T {
	if len(raw) == 0 {
		return def
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return def
	}
	return v
}
