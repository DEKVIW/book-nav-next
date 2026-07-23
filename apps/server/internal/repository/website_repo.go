package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type WebsiteRepo struct {
	db *sql.DB
}

func NewWebsiteRepo(db *sql.DB) *WebsiteRepo {
	return &WebsiteRepo{db: db}
}

func (r *WebsiteRepo) Create(ctx context.Context, w *domain.Website) error {
	now := clock.NowRFC3339()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO websites(title, url, description, icon, category_id, created_by, is_featured, is_private, sort_order, views, is_valid, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		w.Title, w.URL, w.Description, w.Icon, w.CategoryID, w.CreatedBy,
		boolToInt(w.IsFeatured), boolToInt(w.IsPrivate), w.SortOrder, w.Views, boolToInt(w.IsValid), now, now,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	w.ID = id
	w.CreatedAt = parseTime(now)
	w.UpdatedAt = w.CreatedAt
	if err := r.ReplaceViewers(ctx, w.ID, w.ViewerIDs); err != nil {
		return err
	}
	return nil
}

func (r *WebsiteRepo) Update(ctx context.Context, w *domain.Website) error {
	now := clock.NowRFC3339()
	_, err := r.db.ExecContext(ctx, `
UPDATE websites SET title=?, url=?, description=?, icon=?, category_id=?, is_featured=?, is_private=?, sort_order=?, is_valid=?, updated_at=?
WHERE id=?`,
		w.Title, w.URL, w.Description, w.Icon, w.CategoryID,
		boolToInt(w.IsFeatured), boolToInt(w.IsPrivate), w.SortOrder, boolToInt(w.IsValid), now, w.ID,
	)
	if err != nil {
		return err
	}
	w.UpdatedAt = parseTime(now)
	return r.ReplaceViewers(ctx, w.ID, w.ViewerIDs)
}

func (r *WebsiteRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM websites WHERE id = ?`, id)
	return err
}

func (r *WebsiteRepo) DeleteMany(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	q := fmt.Sprintf(`DELETE FROM websites WHERE id IN (%s)`, strings.Join(placeholders, ","))
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *WebsiteRepo) GetByID(ctx context.Context, id int64) (*domain.Website, error) {
	row := r.db.QueryRowContext(ctx, websiteSelect+` WHERE w.id = ?`, id)
	w, err := scanWebsite(row)
	if err != nil || w == nil {
		return w, err
	}
	viewers, err := r.ListViewers(ctx, w.ID)
	if err != nil {
		return nil, err
	}
	w.ViewerIDs = viewers
	return w, nil
}

func (r *WebsiteRepo) FindByURL(ctx context.Context, url string) (*domain.Website, error) {
	row := r.db.QueryRowContext(ctx, websiteSelect+` WHERE w.url = ? LIMIT 1`, url)
	return scanWebsite(row)
}

const websiteSelect = `
SELECT w.id, w.title, w.url, w.description, w.icon, w.category_id, w.created_by,
       w.is_featured, w.is_private, w.sort_order, w.views, w.is_valid, w.created_at, w.updated_at,
       COALESCE(c.name, '')
FROM websites w
LEFT JOIN categories c ON c.id = w.category_id`

func (r *WebsiteRepo) ListByCategory(ctx context.Context, categoryID *int64, limit int) ([]domain.Website, error) {
	var rows *sql.Rows
	var err error
	if categoryID == nil {
		rows, err = r.db.QueryContext(ctx, websiteSelect+`
WHERE w.category_id IS NULL
ORDER BY w.sort_order DESC, w.created_at ASC, w.views DESC
LIMIT ?`, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, websiteSelect+`
WHERE w.category_id = ?
ORDER BY w.sort_order DESC, w.created_at ASC, w.views DESC
LIMIT ?`, *categoryID, limit)
	}
	if err != nil {
		return nil, err
	}
	return collectWebsites(rows)
}

func (r *WebsiteRepo) ListAllByCategory(ctx context.Context, categoryID int64) ([]domain.Website, error) {
	rows, err := r.db.QueryContext(ctx, websiteSelect+`
WHERE w.category_id = ?
ORDER BY w.sort_order DESC, w.created_at ASC, w.views DESC`, categoryID)
	if err != nil {
		return nil, err
	}
	return collectWebsites(rows)
}

func (r *WebsiteRepo) ListFeatured(ctx context.Context, limit int) ([]domain.Website, error) {
	rows, err := r.db.QueryContext(ctx, websiteSelect+`
WHERE w.is_featured = 1
ORDER BY w.views DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	return collectWebsites(rows)
}

func (r *WebsiteRepo) ListAdmin(ctx context.Context, page, pageSize int, categoryID *int64, q string) ([]domain.Website, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	where := []string{"1=1"}
	args := []any{}
	if categoryID != nil {
		where = append(where, "w.category_id = ?")
		args = append(args, *categoryID)
	}
	if strings.TrimSpace(q) != "" {
		where = append(where, "(w.title LIKE ? OR w.url LIKE ? OR w.description LIKE ?)")
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like, like)
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	countQ := `SELECT COUNT(1) FROM websites w WHERE ` + whereSQL
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	args2 := append(append([]any{}, args...), pageSize, offset)
	rows, err := r.db.QueryContext(ctx, websiteSelect+` WHERE `+whereSQL+`
ORDER BY w.created_at DESC LIMIT ? OFFSET ?`, args2...)
	if err != nil {
		return nil, 0, err
	}
	items, err := collectWebsites(rows)
	return items, total, err
}

func (r *WebsiteRepo) ListAll(ctx context.Context) ([]domain.Website, error) {
	rows, err := r.db.QueryContext(ctx, websiteSelect+` ORDER BY w.id ASC`)
	if err != nil {
		return nil, err
	}
	return collectWebsites(rows)
}

func (r *WebsiteRepo) Search(ctx context.Context, query string, limit int) ([]domain.Website, error) {
	like := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, websiteSelect+`
WHERE w.title LIKE ? OR w.description LIKE ? OR w.url LIKE ?
ORDER BY w.views DESC
LIMIT ?`, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	return collectWebsites(rows)
}

// GetByIDs returns websites for the given IDs (order not preserved).
func (r *WebsiteRepo) GetByIDs(ctx context.Context, ids []int64) ([]domain.Website, error) {
	if len(ids) == 0 {
		return []domain.Website{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	q := websiteSelect + ` WHERE w.id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return collectWebsites(rows)
}

func (r *WebsiteRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM websites`).Scan(&n)
	return n, err
}

func (r *WebsiteRepo) CountByCategory(ctx context.Context, categoryID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM websites WHERE category_id = ?`, categoryID).Scan(&n)
	return n, err
}

func (r *WebsiteRepo) IncrementViews(ctx context.Context, id int64) error {
	now := clock.NowRFC3339()
	_, err := r.db.ExecContext(ctx, `
UPDATE websites SET views = views + 1, views_today = views_today + 1, last_viewed_at = ? WHERE id = ?`, now, id)
	return err
}

func (r *WebsiteRepo) UpdateOrders(ctx context.Context, categoryID *int64, ids []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	n := len(ids)
	for i, id := range ids {
		order := n - i
		if _, err := tx.ExecContext(ctx, `UPDATE websites SET sort_order = ?, updated_at = ? WHERE id = ?`, order, clock.NowRFC3339(), id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *WebsiteRepo) UpdateValidity(ctx context.Context, id int64, valid bool, checkedAt string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE websites SET is_valid = ?, last_checked_at = ? WHERE id = ?`, boolToInt(valid), checkedAt, id)
	return err
}

func (r *WebsiteRepo) MaxSortOrder(ctx context.Context, categoryID *int64) (int, error) {
	var n sql.NullInt64
	var err error
	if categoryID == nil {
		err = r.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM websites WHERE category_id IS NULL`).Scan(&n)
	} else {
		err = r.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM websites WHERE category_id = ?`, *categoryID).Scan(&n)
	}
	if err != nil {
		return 0, err
	}
	if !n.Valid {
		return 0, nil
	}
	return int(n.Int64), nil
}

func (r *WebsiteRepo) ListViewers(ctx context.Context, websiteID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id FROM website_viewers WHERE website_id = ?`, websiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *WebsiteRepo) ReplaceViewers(ctx context.Context, websiteID int64, userIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM website_viewers WHERE website_id = ?`, websiteID); err != nil {
		return err
	}
	for _, uid := range userIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO website_viewers(website_id, user_id) VALUES(?,?)`, websiteID, uid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *WebsiteRepo) ClearAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM websites`)
	return err
}

func collectWebsites(rows *sql.Rows) ([]domain.Website, error) {
	defer rows.Close()
	var out []domain.Website
	for rows.Next() {
		w, err := scanWebsite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func scanWebsite(s scannable) (*domain.Website, error) {
	var w domain.Website
	var catID sql.NullInt64
	var featured, private, valid int
	var created, updated string
	var catName string
	if err := s.Scan(
		&w.ID, &w.Title, &w.URL, &w.Description, &w.Icon, &catID, &w.CreatedBy,
		&featured, &private, &w.SortOrder, &w.Views, &valid, &created, &updated, &catName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	w.CategoryID = nullInt64(catID)
	w.IsFeatured = featured != 0
	w.IsPrivate = private != 0
	w.IsValid = valid != 0
	w.CreatedAt = parseTime(created)
	w.UpdatedAt = parseTime(updated)
	w.CategoryName = catName
	return &w, nil
}
