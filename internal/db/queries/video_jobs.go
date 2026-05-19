package queries

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VideoJob struct {
	ID           string
	RequestHash  string
	UserID       string
	APIKeyID     string
	Model        string
	Status       string
	RequestJSON  json.RawMessage
	ResultJSON   json.RawMessage
	ErrorType    *string
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	FinishedAt   *time.Time
}

type VideoJobQueries struct {
	pool *pgxpool.Pool
}

func NewVideoJobQueries(pool *pgxpool.Pool) *VideoJobQueries {
	return &VideoJobQueries{pool: pool}
}

func (q *VideoJobQueries) GetOrCreate(ctx context.Context, j *VideoJob) (*VideoJob, bool, error) {
	inserted := &VideoJob{}
	err := q.pool.QueryRow(ctx, `
		INSERT INTO video_generation_jobs (id, request_hash, user_id, api_key_id, model, status, request_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (request_hash) WHERE status IN ('queued', 'running', 'completed') DO NOTHING
		RETURNING id, request_hash, user_id::text, COALESCE(api_key_id::text, ''), model, status, request_json, COALESCE(result_json, '{}'::jsonb),
			error_type, error_message, created_at, updated_at, finished_at`,
		j.ID, j.RequestHash, j.UserID, nullIfEmpty(j.APIKeyID), j.Model, j.Status, j.RequestJSON,
	).Scan(&inserted.ID, &inserted.RequestHash, &inserted.UserID, &inserted.APIKeyID, &inserted.Model,
		&inserted.Status, &inserted.RequestJSON, &inserted.ResultJSON, &inserted.ErrorType,
		&inserted.ErrorMessage, &inserted.CreatedAt, &inserted.UpdatedAt, &inserted.FinishedAt)
	if err == nil {
		return inserted, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	existing, err := q.GetByHash(ctx, j.RequestHash)
	return existing, false, err
}

func (q *VideoJobQueries) GetByID(ctx context.Context, id string) (*VideoJob, error) {
	return q.scanOne(ctx, `WHERE id=$1`, id)
}

func (q *VideoJobQueries) GetByHash(ctx context.Context, hash string) (*VideoJob, error) {
	return q.scanOne(ctx, `WHERE request_hash=$1 AND status IN ('queued', 'running', 'completed') ORDER BY created_at DESC LIMIT 1`, hash)
}

func (q *VideoJobQueries) MarkRunning(ctx context.Context, id string) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE video_generation_jobs SET status='running', updated_at=now()
		WHERE id=$1 AND status='queued'`, id)
	return err
}

func (q *VideoJobQueries) Complete(ctx context.Context, id string, result json.RawMessage) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE video_generation_jobs
		SET status='completed', result_json=$2, error_type=NULL, error_message=NULL,
			updated_at=now(), finished_at=now()
		WHERE id=$1`, id, result)
	return err
}

func (q *VideoJobQueries) Fail(ctx context.Context, id, errorType, message string) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE video_generation_jobs
		SET status='failed', error_type=$2, error_message=$3, updated_at=now(), finished_at=now()
		WHERE id=$1`, id, errorType, message)
	return err
}

func (q *VideoJobQueries) scanOne(ctx context.Context, where string, args ...any) (*VideoJob, error) {
	j := &VideoJob{}
	err := q.pool.QueryRow(ctx, `
		SELECT id, request_hash, user_id::text, COALESCE(api_key_id::text, ''), model, status, request_json,
			COALESCE(result_json, '{}'::jsonb), error_type, error_message, created_at, updated_at, finished_at
		FROM video_generation_jobs `+where, args...,
	).Scan(&j.ID, &j.RequestHash, &j.UserID, &j.APIKeyID, &j.Model, &j.Status, &j.RequestJSON,
		&j.ResultJSON, &j.ErrorType, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
