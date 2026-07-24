package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/clock"
)

type JobRepo struct {
	db *sql.DB
}

func NewJobRepo(db *sql.DB) *JobRepo {
	return &JobRepo{db: db}
}

func (r *JobRepo) Create(ctx context.Context, j *domain.Job) error {
	now := clock.NowRFC3339()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO jobs(type, status, progress, total, success, failed, payload_json, result_json, error, created_by, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		j.Type, j.Status, j.Progress, j.Total, j.Success, j.Failed, j.PayloadJSON, j.ResultJSON, j.Error, j.CreatedBy, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	j.ID = id
	j.CreatedAt = parseTime(now)
	j.UpdatedAt = j.CreatedAt
	return nil
}

func (r *JobRepo) Get(ctx context.Context, id int64) (*domain.Job, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, type, status, progress, total, success, failed, payload_json, result_json, error, created_by, started_at, finished_at, created_at, updated_at
FROM jobs WHERE id = ?`, id)
	return scanJob(row)
}

func (r *JobRepo) List(ctx context.Context, limit int) ([]domain.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, type, status, progress, total, success, failed, payload_json, result_json, error, created_by, started_at, finished_at, created_at, updated_at
FROM jobs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func (r *JobRepo) Update(ctx context.Context, j *domain.Job) error {
	now := clock.NowRFC3339()
	var started, finished any
	if j.StartedAt != nil {
		started = j.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if j.FinishedAt != nil {
		finished = j.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	// Do not overwrite a cancelled job with a stale running status from a worker race.
	_, err := r.db.ExecContext(ctx, `
UPDATE jobs SET
  status = CASE WHEN status = 'cancelled' AND ? != 'cancelled' THEN status ELSE ? END,
  progress=?, total=?, success=?, failed=?, payload_json=?, result_json=?,
  error = CASE WHEN status = 'cancelled' AND ? != 'cancelled' THEN error ELSE ? END,
  started_at=?,
  finished_at = CASE WHEN status = 'cancelled' AND ? != 'cancelled' THEN finished_at ELSE ? END,
  updated_at=?
WHERE id=?`,
		j.Status, j.Status,
		j.Progress, j.Total, j.Success, j.Failed, j.PayloadJSON, j.ResultJSON,
		j.Status, j.Error,
		started,
		j.Status, finished,
		now, j.ID,
	)
	j.UpdatedAt = parseTime(now)
	return err
}

func (r *JobRepo) CountRunning(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM jobs WHERE status IN ('pending','running')`).Scan(&n)
	return n, err
}

// CountActiveByType counts pending/running jobs of a given type (anti-duplicate starts).
func (r *JobRepo) CountActiveByType(ctx context.Context, jobType string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM jobs WHERE type = ? AND status IN ('pending','running')`, jobType,
	).Scan(&n)
	return n, err
}

func (r *JobRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = ? AND status NOT IN ('pending','running')`, id)
	return err
}

func (r *JobRepo) DeleteFinished(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM jobs WHERE status IN ('completed','failed','cancelled')`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *JobRepo) CountFinished(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM jobs WHERE status IN ('completed','failed','cancelled')`,
	).Scan(&n)
	return n, err
}

func scanJob(s scannable) (*domain.Job, error) {
	var j domain.Job
	var createdBy sql.NullInt64
	var started, finished sql.NullString
	var created, updated string
	if err := s.Scan(
		&j.ID, &j.Type, &j.Status, &j.Progress, &j.Total, &j.Success, &j.Failed,
		&j.PayloadJSON, &j.ResultJSON, &j.Error, &createdBy, &started, &finished, &created, &updated,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	j.CreatedBy = nullInt64(createdBy)
	j.StartedAt = parseTimePtr(started)
	j.FinishedAt = parseTimePtr(finished)
	j.CreatedAt = parseTime(created)
	j.UpdatedAt = parseTime(updated)
	return &j, nil
}
