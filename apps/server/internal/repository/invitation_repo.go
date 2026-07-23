package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type InvitationRepo struct {
	db *sql.DB
}

func NewInvitationRepo(db *sql.DB) *InvitationRepo {
	return &InvitationRepo{db: db}
}

func (r *InvitationRepo) Create(ctx context.Context, code string, createdBy *int64) (*domain.InvitationCode, error) {
	now := clock.NowRFC3339()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO invitation_codes(code, created_by, is_active, created_at) VALUES(?,?,1,?)`,
		code, createdBy, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &domain.InvitationCode{
		ID:        id,
		Code:      code,
		CreatedBy: createdBy,
		IsActive:  true,
		CreatedAt: parseTime(now),
	}, nil
}

func (r *InvitationRepo) GetByCode(ctx context.Context, code string) (*domain.InvitationCode, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, code, created_by, used_by, used_at, is_active, created_at FROM invitation_codes WHERE code = ?`, code)
	return scanInvite(row)
}

func (r *InvitationRepo) List(ctx context.Context) ([]domain.InvitationCode, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, code, created_by, used_by, used_at, is_active, created_at FROM invitation_codes ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.InvitationCode
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

func (r *InvitationRepo) MarkUsed(ctx context.Context, id, userID int64) error {
	now := clock.NowRFC3339()
	_, err := r.db.ExecContext(ctx, `
UPDATE invitation_codes SET used_by=?, used_at=?, is_active=0 WHERE id=? AND is_active=1 AND used_by IS NULL`,
		userID, now, id,
	)
	return err
}

func (r *InvitationRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM invitation_codes WHERE id = ?`, id)
	return err
}

func (r *InvitationRepo) CountActive(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM invitation_codes WHERE is_active=1 AND used_by IS NULL`).Scan(&n)
	return n, err
}

func scanInvite(s scannable) (*domain.InvitationCode, error) {
	var inv domain.InvitationCode
	var createdBy, usedBy sql.NullInt64
	var usedAt sql.NullString
	var active int
	var created string
	if err := s.Scan(&inv.ID, &inv.Code, &createdBy, &usedBy, &usedAt, &active, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	inv.CreatedBy = nullInt64(createdBy)
	inv.UsedBy = nullInt64(usedBy)
	inv.UsedAt = parseTimePtr(usedAt)
	inv.IsActive = active != 0
	inv.CreatedAt = parseTime(created)
	return &inv, nil
}
