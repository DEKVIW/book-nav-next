package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type SearchShareRepo struct {
	db *sql.DB
}

func NewSearchShareRepo(db *sql.DB) *SearchShareRepo {
	return &SearchShareRepo{db: db}
}

func (r *SearchShareRepo) Create(ctx context.Context, share *domain.SearchShare, websiteIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := clock.NowRFC3339()
	result, err := tx.ExecContext(ctx, `
INSERT INTO search_shares(token, created_by, creator_type, title, query, mode, summary, status,
                          include_private, created_ip_hash, expires_at, view_count, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		share.Token, share.CreatedBy, share.CreatorType, share.Title, share.Query, share.Mode, share.Summary,
		share.Status, boolToInt(share.IncludePrivate), share.CreatedIPHash, formatTimePtr(share.ExpiresAt), 0, now, now,
	)
	if err != nil {
		return err
	}
	share.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	share.CreatedAt = parseTime(now)
	share.UpdatedAt = share.CreatedAt
	share.ItemCount = len(websiteIDs)

	for order, websiteID := range websiteIDs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO search_share_items(share_id, website_id, sort_order) VALUES(?,?,?)`,
			share.ID, websiteID, order,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SearchShareRepo) GetByToken(ctx context.Context, token string) (*domain.SearchShare, error) {
	return r.get(ctx, `WHERE s.token = ?`, token)
}

func (r *SearchShareRepo) GetByID(ctx context.Context, id int64) (*domain.SearchShare, error) {
	return r.get(ctx, `WHERE s.id = ?`, id)
}

func (r *SearchShareRepo) get(ctx context.Context, where string, args ...any) (*domain.SearchShare, error) {
	row := r.db.QueryRowContext(ctx, `
	SELECT s.id, s.token, s.created_by, COALESCE(u.username, ''), s.creator_type, s.title, s.query, s.mode,
	       s.summary, s.status, s.include_private, s.expires_at, s.view_count, COUNT(i.website_id),
       s.created_at, s.updated_at
FROM search_shares s
LEFT JOIN users u ON u.id = s.created_by
LEFT JOIN search_share_items i ON i.share_id = s.id
`+where+`
GROUP BY s.id`, args...)
	return scanSearchShare(row)
}

func (r *SearchShareRepo) List(ctx context.Context, page, pageSize int, status, query string) ([]domain.SearchShare, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	where := []string{"(s.expires_at IS NULL OR s.expires_at > ?)"}
	args := []any{clock.NowRFC3339()}
	if status != "" {
		where = append(where, "s.status = ?")
		args = append(args, status)
	}
	if query = strings.TrimSpace(query); query != "" {
		where = append(where, "(s.title LIKE ? OR s.query LIKE ?)")
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM search_shares s WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
	SELECT s.id, s.token, s.created_by, COALESCE(u.username, ''), s.creator_type, s.title, s.query, s.mode,
	       s.summary, s.status, s.include_private, s.expires_at, s.view_count, COUNT(i.website_id),
       s.created_at, s.updated_at
FROM search_shares s
LEFT JOIN users u ON u.id = s.created_by
LEFT JOIN search_share_items i ON i.share_id = s.id
WHERE `+whereSQL+`
GROUP BY s.id
ORDER BY s.created_at DESC
LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]domain.SearchShare, 0)
	for rows.Next() {
		share, err := scanSearchShare(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *share)
	}
	return items, total, rows.Err()
}

func (r *SearchShareRepo) ListItems(ctx context.Context, shareID int64) ([]domain.SearchShareItem, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT share_id, website_id, sort_order
FROM search_share_items
WHERE share_id = ?
ORDER BY sort_order ASC`, shareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.SearchShareItem, 0)
	for rows.Next() {
		var item domain.SearchShareItem
		if err := rows.Scan(&item.ShareID, &item.WebsiteID, &item.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SearchShareRepo) IncrementViews(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE search_shares SET view_count = view_count + 1 WHERE id = ?`, id)
	return err
}

func (r *SearchShareRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
DELETE FROM search_shares
WHERE expires_at IS NOT NULL AND expires_at <= ?`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *SearchShareRepo) CountGuestCreatedSince(ctx context.Context, ipHash string, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(1) FROM search_shares
WHERE creator_type = 'guest' AND created_ip_hash = ? AND created_at >= ?`,
		ipHash, since.UTC().Format(time.RFC3339Nano)).Scan(&count)
	return count, err
}

func (r *SearchShareRepo) CountActiveGuestByIP(ctx context.Context, ipHash string, now time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(1) FROM search_shares
WHERE creator_type = 'guest' AND created_ip_hash = ? AND status = ?
  AND (expires_at IS NULL OR expires_at > ?)`,
		ipHash, domain.SearchShareStatusActive, now.UTC().Format(time.RFC3339Nano)).Scan(&count)
	return count, err
}

func (r *SearchShareRepo) Revoke(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE search_shares SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
		domain.SearchShareStatusRevoked, clock.NowRFC3339(), id, domain.SearchShareStatusActive)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SearchShareRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM search_shares WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type searchShareScanner interface {
	Scan(dest ...any) error
}

func scanSearchShare(s searchShareScanner) (*domain.SearchShare, error) {
	var share domain.SearchShare
	var createdBy sql.NullInt64
	var includePrivate int
	var expires, created, updated sql.NullString
	if err := s.Scan(
		&share.ID, &share.Token, &createdBy, &share.CreatorName, &share.CreatorType,
		&share.Title, &share.Query, &share.Mode, &share.Summary, &share.Status,
		&includePrivate, &expires, &share.ViewCount, &share.ItemCount, &created, &updated,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if createdBy.Valid {
		value := createdBy.Int64
		share.CreatedBy = &value
	}
	share.IncludePrivate = includePrivate != 0
	share.ExpiresAt = parseTimePtr(expires)
	share.CreatedAt = parseTime(created.String)
	share.UpdatedAt = parseTime(updated.String)
	return &share, nil
}

func formatTimePtr(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
