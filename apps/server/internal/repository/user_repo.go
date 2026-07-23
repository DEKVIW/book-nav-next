package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	now := clock.NowRFC3339()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO users(username, email, password_hash, avatar, role, created_at, updated_at)
VALUES(?,?,?,?,?,?,?)`,
		u.Username, u.Email, u.PasswordHash, u.Avatar, string(u.Role), now, now,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	u.CreatedAt = parseTime(now)
	u.UpdatedAt = u.CreatedAt
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT id, username, email, password_hash, avatar, role, created_at, updated_at FROM users WHERE id = ?`, id)
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT id, username, email, password_hash, avatar, role, created_at, updated_at FROM users WHERE username = ?`, username)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanOne(ctx, `SELECT id, username, email, password_hash, avatar, role, created_at, updated_at FROM users WHERE email = ?`, email)
}

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&n)
	return n, err
}

func (r *UserRepo) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, username, email, password_hash, avatar, role, created_at, updated_at FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	now := clock.NowRFC3339()
	_, err := r.db.ExecContext(ctx, `
UPDATE users SET username=?, email=?, password_hash=?, avatar=?, role=?, updated_at=? WHERE id=?`,
		u.Username, u.Email, u.PasswordHash, u.Avatar, string(u.Role), now, u.ID,
	)
	return err
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *UserRepo) scanOne(ctx context.Context, q string, args ...any) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, q, args...)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanUser(s scannable) (*domain.User, error) {
	var u domain.User
	var role string
	var created, updated string
	if err := s.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Avatar, &role, &created, &updated); err != nil {
		return nil, err
	}
	u.Role = domain.Role(role)
	u.CreatedAt = parseTime(created)
	u.UpdatedAt = parseTime(updated)
	return &u, nil
}

func (r *UserRepo) EnsureSuperAdmin(ctx context.Context, username, email, passwordHash string) error {
	existing, err := r.GetByUsername(ctx, username)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.Role != domain.RoleSuperAdmin {
			existing.Role = domain.RoleSuperAdmin
			return r.Update(ctx, existing)
		}
		return nil
	}
	byEmail, err := r.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	if byEmail != nil {
		return fmt.Errorf("admin email already used by another user")
	}
	u := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         domain.RoleSuperAdmin,
	}
	return r.Create(ctx, u)
}
