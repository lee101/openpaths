package queries

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ResearchJob struct {
	ID           string
	UserID       string
	APIKeyID     string
	Query        string
	Depth        string
	Model        string
	Status       string
	Steps        json.RawMessage
	Report       string
	Citations    json.RawMessage
	CostUnits    int64
	ErrorType    *string
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	FinishedAt   *time.Time
}

type ResearchJobQueries struct {
	pool *pgxpool.Pool
}

func NewResearchJobQueries(pool *pgxpool.Pool) *ResearchJobQueries {
	return &ResearchJobQueries{pool: pool}
}

const researchJobCols = `id, user_id::text, COALESCE(api_key_id::text, ''), query, depth, model, status,
	steps, report, citations, cost_units, error_type, error_message, created_at, updated_at, finished_at`

func (q *ResearchJobQueries) Create(ctx context.Context, j *ResearchJob) (*ResearchJob, error) {
	if j.Depth == "" {
		j.Depth = "standard"
	}
	if j.Status == "" {
		j.Status = "queued"
	}
	return q.scanOne(ctx, `
		INSERT INTO research_jobs (id, user_id, api_key_id, query, depth, model, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING `+researchJobCols,
		j.ID, j.UserID, nullIfEmpty(j.APIKeyID), j.Query, j.Depth, j.Model, j.Status)
}

func (q *ResearchJobQueries) GetByID(ctx context.Context, id string) (*ResearchJob, error) {
	return q.scanOne(ctx, `SELECT `+researchJobCols+` FROM research_jobs WHERE id=$1`, id)
}

func (q *ResearchJobQueries) ListByUser(ctx context.Context, userID string, limit int) ([]*ResearchJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := q.pool.Query(ctx, `SELECT `+researchJobCols+`
		FROM research_jobs WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ResearchJob
	for rows.Next() {
		j, err := scanResearchJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (q *ResearchJobQueries) MarkRunning(ctx context.Context, id string) error {
	_, err := q.pool.Exec(ctx, `UPDATE research_jobs SET status='running', updated_at=now()
		WHERE id=$1 AND status='queued'`, id)
	return err
}

// AppendStep atomically pushes one reasoning step onto the steps array so the
// UI can poll the chain as it grows.
func (q *ResearchJobQueries) AppendStep(ctx context.Context, id string, step json.RawMessage) error {
	_, err := q.pool.Exec(ctx, `UPDATE research_jobs
		SET steps = steps || $2::jsonb, updated_at=now() WHERE id=$1`, id, step)
	return err
}

func (q *ResearchJobQueries) Complete(ctx context.Context, id, report string, citations json.RawMessage, costUnits int64) error {
	if len(citations) == 0 {
		citations = json.RawMessage("[]")
	}
	_, err := q.pool.Exec(ctx, `UPDATE research_jobs
		SET status='completed', report=$2, citations=$3::jsonb, cost_units=$4,
			error_type=NULL, error_message=NULL, updated_at=now(), finished_at=now()
		WHERE id=$1`, id, report, citations, costUnits)
	return err
}

func (q *ResearchJobQueries) Fail(ctx context.Context, id, errorType, message string) error {
	_, err := q.pool.Exec(ctx, `UPDATE research_jobs
		SET status='failed', error_type=$2, error_message=$3, updated_at=now(), finished_at=now()
		WHERE id=$1`, id, errorType, message)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanResearchJob(row rowScanner) (*ResearchJob, error) {
	j := &ResearchJob{}
	err := row.Scan(&j.ID, &j.UserID, &j.APIKeyID, &j.Query, &j.Depth, &j.Model, &j.Status,
		&j.Steps, &j.Report, &j.Citations, &j.CostUnits, &j.ErrorType, &j.ErrorMessage,
		&j.CreatedAt, &j.UpdatedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (q *ResearchJobQueries) scanOne(ctx context.Context, sql string, args ...any) (*ResearchJob, error) {
	return scanResearchJob(q.pool.QueryRow(ctx, sql, args...))
}
