package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type CategoryRepo struct {
	db *sql.DB
}

func NewCategoryRepo(db *sql.DB) *CategoryRepo {
	return &CategoryRepo{db: db}
}

func (r *CategoryRepo) Create(ctx context.Context, c *domain.Category) error {
	now := clock.NowRFC3339()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO categories(name, description, icon, color, sort_order, display_limit, parent_id, created_at)
VALUES(?,?,?,?,?,?,?,?)`,
		c.Name, c.Description, c.Icon, c.Color, c.SortOrder, c.DisplayLimit, c.ParentID, now,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	c.CreatedAt = parseTime(now)
	return nil
}

func (r *CategoryRepo) Update(ctx context.Context, c *domain.Category) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE categories SET name=?, description=?, icon=?, color=?, sort_order=?, display_limit=?, parent_id=?
WHERE id=?`,
		c.Name, c.Description, c.Icon, c.Color, c.SortOrder, c.DisplayLimit, c.ParentID, c.ID,
	)
	return err
}

func (r *CategoryRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	return err
}

func (r *CategoryRepo) GetByID(ctx context.Context, id int64) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, description, icon, color, sort_order, display_limit, parent_id, created_at
FROM categories WHERE id = ?`, id)
	return scanCategory(row)
}

func (r *CategoryRepo) ListAll(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, description, icon, color, sort_order, display_limit, parent_id, created_at
FROM categories ORDER BY sort_order DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *CategoryRepo) CountWebsites(ctx context.Context, categoryID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM websites WHERE category_id = ?`, categoryID).Scan(&n)
	return n, err
}

func (r *CategoryRepo) CountChildren(ctx context.Context, parentID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM categories WHERE parent_id = ?`, parentID).Scan(&n)
	return n, err
}

func (r *CategoryRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM categories`).Scan(&n)
	return n, err
}

// ClearAll removes every category. Call only after websites are gone (or accept SET NULL on website.category_id).
func (r *CategoryRepo) ClearAll(ctx context.Context) error {
	// children first so parent_id FK does not block
	if _, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE parent_id IS NOT NULL`); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories`)
	return err
}

func (r *CategoryRepo) UpdateOrders(ctx context.Context, ids []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// higher index => higher sort_order (DESC listing)
	n := len(ids)
	for i, id := range ids {
		order := n - i
		if _, err := tx.ExecContext(ctx, `UPDATE categories SET sort_order = ? WHERE id = ?`, order, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *CategoryRepo) MaxSortOrder(ctx context.Context, parentID *int64) (int, error) {
	var n sql.NullInt64
	var err error
	if parentID == nil {
		err = r.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM categories WHERE parent_id IS NULL`).Scan(&n)
	} else {
		err = r.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM categories WHERE parent_id = ?`, *parentID).Scan(&n)
	}
	if err != nil {
		return 0, err
	}
	if !n.Valid {
		return 0, nil
	}
	return int(n.Int64), nil
}

func scanCategory(s scannable) (*domain.Category, error) {
	var c domain.Category
	var parent sql.NullInt64
	var created string
	if err := s.Scan(&c.ID, &c.Name, &c.Description, &c.Icon, &c.Color, &c.SortOrder, &c.DisplayLimit, &parent, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	c.ParentID = nullInt64(parent)
	c.CreatedAt = parseTime(created)
	return &c, nil
}
