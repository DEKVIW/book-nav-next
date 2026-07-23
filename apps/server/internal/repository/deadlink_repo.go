package repository

import (
	"context"
	"database/sql"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type DeadlinkRepo struct {
	db *sql.DB
}

func NewDeadlinkRepo(db *sql.DB) *DeadlinkRepo {
	return &DeadlinkRepo{db: db}
}

func (r *DeadlinkRepo) Create(ctx context.Context, d *domain.DeadlinkCheck) error {
	now := clock.NowRFC3339()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO deadlink_checks(batch_id, website_id, url, is_valid, status_code, error_type, error_message, response_time_ms, checked_at)
VALUES(?,?,?,?,?,?,?,?,?)`,
		d.BatchID, d.WebsiteID, d.URL, boolToInt(d.IsValid), d.StatusCode, d.ErrorType, d.ErrorMessage, d.ResponseTimeMs, now,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	d.ID = id
	d.CheckedAt = parseTime(now)
	return nil
}

func (r *DeadlinkRepo) ListByBatch(ctx context.Context, batchID string, invalidOnly bool) ([]domain.DeadlinkCheck, error) {
	q := `
SELECT d.id, d.batch_id, d.website_id, d.url, d.is_valid, d.status_code, d.error_type, d.error_message, d.response_time_ms, d.checked_at,
       COALESCE(w.title,'')
FROM deadlink_checks d
LEFT JOIN websites w ON w.id = d.website_id
WHERE d.batch_id = ?`
	if invalidOnly {
		q += ` AND d.is_valid = 0`
	}
	q += ` ORDER BY d.id ASC`
	rows, err := r.db.QueryContext(ctx, q, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DeadlinkCheck
	for rows.Next() {
		var d domain.DeadlinkCheck
		var valid int
		var sc, rtm sql.NullInt64
		var checked string
		if err := rows.Scan(&d.ID, &d.BatchID, &d.WebsiteID, &d.URL, &valid, &sc, &d.ErrorType, &d.ErrorMessage, &rtm, &checked, &d.WebsiteTitle); err != nil {
			return nil, err
		}
		d.IsValid = valid != 0
		if sc.Valid {
			v := int(sc.Int64)
			d.StatusCode = &v
		}
		if rtm.Valid {
			v := int(rtm.Int64)
			d.ResponseTimeMs = &v
		}
		d.CheckedAt = parseTime(checked)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DeadlinkRepo) ClearBatch(ctx context.Context, batchID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM deadlink_checks WHERE batch_id = ?`, batchID)
	return err
}

func (r *DeadlinkRepo) ClearAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM deadlink_checks`)
	return err
}
