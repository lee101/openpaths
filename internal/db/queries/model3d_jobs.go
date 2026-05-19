package queries

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Model3DJob struct {
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

type Model3DJobQueries struct {
	pool *pgxpool.Pool
}

func NewModel3DJobQueries(pool *pgxpool.Pool) *Model3DJobQueries {
	return &Model3DJobQueries{pool: pool}
}

func (q *Model3DJobQueries) GetOrCreate(ctx context.Context, j *Model3DJob) (*Model3DJob, bool, error) {
	inserted := &Model3DJob{}
	err := q.pool.QueryRow(ctx, `
		INSERT INTO model3d_generation_jobs (id, request_hash, user_id, api_key_id, model, status, request_json)
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

func (q *Model3DJobQueries) GetByID(ctx context.Context, id string) (*Model3DJob, error) {
	return q.scanOne(ctx, `WHERE id=$1`, id)
}

func (q *Model3DJobQueries) GetByHash(ctx context.Context, hash string) (*Model3DJob, error) {
	return q.scanOne(ctx, `WHERE request_hash=$1 AND status IN ('queued', 'running', 'completed') ORDER BY created_at DESC LIMIT 1`, hash)
}

func (q *Model3DJobQueries) MarkRunning(ctx context.Context, id string) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE model3d_generation_jobs SET status='running', updated_at=now()
		WHERE id=$1 AND status='queued'`, id)
	return err
}

func (q *Model3DJobQueries) Complete(ctx context.Context, id string, result json.RawMessage) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE model3d_generation_jobs
		SET status='completed', result_json=$2, error_type=NULL, error_message=NULL,
			updated_at=now(), finished_at=now()
		WHERE id=$1`, id, result)
	return err
}

func (q *Model3DJobQueries) Fail(ctx context.Context, id, errorType, message string) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE model3d_generation_jobs
		SET status='failed', error_type=$2, error_message=$3, updated_at=now(), finished_at=now()
		WHERE id=$1`, id, errorType, message)
	return err
}

func (q *Model3DJobQueries) scanOne(ctx context.Context, where string, args ...any) (*Model3DJob, error) {
	j := &Model3DJob{}
	err := q.pool.QueryRow(ctx, `
		SELECT id, request_hash, user_id::text, COALESCE(api_key_id::text, ''), model, status, request_json,
			COALESCE(result_json, '{}'::jsonb), error_type, error_message, created_at, updated_at, finished_at
		FROM model3d_generation_jobs `+where, args...,
	).Scan(&j.ID, &j.RequestHash, &j.UserID, &j.APIKeyID, &j.Model, &j.Status, &j.RequestJSON,
		&j.ResultJSON, &j.ErrorType, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}
