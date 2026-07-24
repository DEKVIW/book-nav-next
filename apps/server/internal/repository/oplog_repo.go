package repository

import (
	"context"
	"database/sql"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type OpLogRepo struct {
	db *sql.DB
}

func NewOpLogRepo(db *sql.DB) *OpLogRepo {
	return &OpLogRepo{db: db}
}

func (r *OpLogRepo) Create(ctx context.Context, log *domain.OperationLog) error {
	now := clock.NowRFC3339()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO operation_logs(user_id, action, website_id, website_title, website_url, website_icon, category_id, category_name, details_json, created_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
		log.UserID, log.Action, log.WebsiteID, log.WebsiteTitle, log.WebsiteURL, log.WebsiteIcon,
		log.CategoryID, log.CategoryName, log.DetailsJSON, now,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	log.ID = id
	log.CreatedAt = parseTime(now)
	return nil
}

func (r *OpLogRepo) List(ctx context.Context, page, pageSize int, userID *int64) ([]domain.OperationLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	where := "1=1"
	args := []any{}
	if userID != nil {
		where = "user_id = ?"
		args = append(args, *userID)
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM operation_logs WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args2 := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT id, user_id, action, website_id, website_title, website_url, website_icon, category_id, category_name, details_json, created_at
FROM operation_logs WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.OperationLog
	for rows.Next() {
		var l domain.OperationLog
		var uid, wid, cid sql.NullInt64
		var created string
		if err := rows.Scan(&l.ID, &uid, &l.Action, &wid, &l.WebsiteTitle, &l.WebsiteURL, &l.WebsiteIcon, &cid, &l.CategoryName, &l.DetailsJSON, &created); err != nil {
			return nil, 0, err
		}
		l.UserID = nullInt64(uid)
		l.WebsiteID = nullInt64(wid)
		l.CategoryID = nullInt64(cid)
		l.CreatedAt = parseTime(created)
		out = append(out, l)
	}
	return out, total, rows.Err()
}

func (r *OpLogRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM operation_logs WHERE id = ?`, id)
	return err
}

func (r *OpLogRepo) DeleteMany(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *OpLogRepo) ClearUser(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM operation_logs WHERE user_id = ?`, userID)
	return err
}

func (r *OpLogRepo) ClearAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM operation_logs`)
	return err
}

func (r *OpLogRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM operation_logs`).Scan(&n)
	return n, err
}
