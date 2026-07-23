package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	now := clock.NowRFC3339()
	_, err := r.db.ExecContext(ctx, `
INSERT INTO sessions(id, user_id, csrf_token, expires_at, created_at)
VALUES(?,?,?,?,?)`,
		s.ID, s.UserID, s.CSRFToken, s.ExpiresAt.UTC().Format(time.RFC3339Nano), now,
	)
	s.CreatedAt = parseTime(now)
	return err
}

func (r *SessionRepo) Get(ctx context.Context, id string) (*domain.Session, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, user_id, csrf_token, expires_at, created_at FROM sessions WHERE id = ?`, id)
	var s domain.Session
	var exp, created string
	if err := row.Scan(&s.ID, &s.UserID, &s.CSRFToken, &exp, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s.ExpiresAt = parseTime(exp)
	s.CreatedAt = parseTime(created)
	return &s, nil
}

func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	now := clock.NowRFC3339()
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now)
	return err
}

func (r *SessionRepo) DeleteByUser(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}
