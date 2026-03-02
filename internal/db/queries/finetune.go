package queries

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FineTuneJob struct {
	ID              string
	UserID          string
	Provider        string
	ProviderJobID   string
	Model           string
	FineTunedModel  *string
	Status          string
	TrainingFile    string
	ValidationFile  *string
	Hyperparameters json.RawMessage
	ResultFiles     json.RawMessage
	TrainedTokens   int64
	ErrorMessage    *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	FinishedAt      *time.Time
}

type FineTuneEvent struct {
	ID        string
	JobID     string
	Type      string
	Level     string
	Message   string
	Data      json.RawMessage
	CreatedAt time.Time
}

type FineTuneFile struct {
	ID             string
	UserID         string
	Provider       string
	ProviderFileID string
	Filename       string
	Purpose        string
	Bytes          int64
	StorageURL     string
	CreatedAt      time.Time
}

type FineTuneQueries struct {
	pool *pgxpool.Pool
}

func NewFineTuneQueries(pool *pgxpool.Pool) *FineTuneQueries {
	return &FineTuneQueries{pool: pool}
}

func (q *FineTuneQueries) CreateJob(ctx context.Context, j *FineTuneJob) error {
	return q.pool.QueryRow(ctx, `
		INSERT INTO fine_tuning_jobs (user_id, provider, provider_job_id, model, status,
			training_file, validation_file, hyperparameters)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at`,
		j.UserID, j.Provider, j.ProviderJobID, j.Model, j.Status,
		j.TrainingFile, j.ValidationFile, j.Hyperparameters,
	).Scan(&j.ID, &j.CreatedAt)
}

func (q *FineTuneQueries) GetJob(ctx context.Context, jobID string) (*FineTuneJob, error) {
	j := &FineTuneJob{}
	err := q.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_job_id, model, fine_tuned_model, status,
			training_file, validation_file, hyperparameters, result_files, trained_tokens,
			error_message, created_at, updated_at, finished_at
		FROM fine_tuning_jobs WHERE id=$1`, jobID,
	).Scan(&j.ID, &j.UserID, &j.Provider, &j.ProviderJobID, &j.Model,
		&j.FineTunedModel, &j.Status, &j.TrainingFile, &j.ValidationFile,
		&j.Hyperparameters, &j.ResultFiles, &j.TrainedTokens,
		&j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (q *FineTuneQueries) ListJobsByUser(ctx context.Context, userID string, limit int) ([]*FineTuneJob, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, provider, provider_job_id, model, fine_tuned_model, status,
			training_file, validation_file, hyperparameters, result_files, trained_tokens,
			error_message, created_at, updated_at, finished_at
		FROM fine_tuning_jobs WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*FineTuneJob
	for rows.Next() {
		j := &FineTuneJob{}
		if err := rows.Scan(&j.ID, &j.UserID, &j.Provider, &j.ProviderJobID, &j.Model,
			&j.FineTunedModel, &j.Status, &j.TrainingFile, &j.ValidationFile,
			&j.Hyperparameters, &j.ResultFiles, &j.TrainedTokens,
			&j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt, &j.FinishedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (q *FineTuneQueries) UpdateJobStatus(ctx context.Context, jobID, status string, fineTunedModel *string, trainedTokens int64, errorMsg *string, finishedAt *time.Time) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE fine_tuning_jobs SET status=$2, fine_tuned_model=$3, trained_tokens=$4,
			error_message=$5, finished_at=$6, updated_at=now()
		WHERE id=$1`, jobID, status, fineTunedModel, trainedTokens, errorMsg, finishedAt)
	return err
}

func (q *FineTuneQueries) UpdateProviderJobID(ctx context.Context, jobID, providerJobID string) error {
	_, err := q.pool.Exec(ctx, `UPDATE fine_tuning_jobs SET provider_job_id=$2, updated_at=now() WHERE id=$1`, jobID, providerJobID)
	return err
}

func (q *FineTuneQueries) AddEvent(ctx context.Context, e *FineTuneEvent) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO fine_tuning_events (job_id, type, level, message, data) VALUES ($1,$2,$3,$4,$5)`,
		e.JobID, e.Type, e.Level, e.Message, e.Data)
	return err
}

func (q *FineTuneQueries) ListEvents(ctx context.Context, jobID string) ([]*FineTuneEvent, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, job_id, type, level, message, data, created_at
		FROM fine_tuning_events WHERE job_id=$1 ORDER BY created_at`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*FineTuneEvent
	for rows.Next() {
		e := &FineTuneEvent{}
		if err := rows.Scan(&e.ID, &e.JobID, &e.Type, &e.Level, &e.Message, &e.Data, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (q *FineTuneQueries) CreateFile(ctx context.Context, f *FineTuneFile) error {
	return q.pool.QueryRow(ctx, `
		INSERT INTO fine_tuning_files (user_id, provider, provider_file_id, filename, purpose, bytes, storage_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`,
		f.UserID, f.Provider, f.ProviderFileID, f.Filename, f.Purpose, f.Bytes, f.StorageURL,
	).Scan(&f.ID, &f.CreatedAt)
}

func (q *FineTuneQueries) GetFile(ctx context.Context, fileID string) (*FineTuneFile, error) {
	f := &FineTuneFile{}
	err := q.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_file_id, filename, purpose, bytes, storage_url, created_at
		FROM fine_tuning_files WHERE id=$1`, fileID,
	).Scan(&f.ID, &f.UserID, &f.Provider, &f.ProviderFileID, &f.Filename, &f.Purpose, &f.Bytes, &f.StorageURL, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (q *FineTuneQueries) ListFilesByUser(ctx context.Context, userID string) ([]*FineTuneFile, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, provider, provider_file_id, filename, purpose, bytes, storage_url, created_at
		FROM fine_tuning_files WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FineTuneFile
	for rows.Next() {
		f := &FineTuneFile{}
		if err := rows.Scan(&f.ID, &f.UserID, &f.Provider, &f.ProviderFileID, &f.Filename, &f.Purpose, &f.Bytes, &f.StorageURL, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (q *FineTuneQueries) GetActiveJobs(ctx context.Context) ([]*FineTuneJob, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, provider, provider_job_id, model, fine_tuned_model, status,
			training_file, validation_file, hyperparameters, result_files, trained_tokens,
			error_message, created_at, updated_at, finished_at
		FROM fine_tuning_jobs WHERE status IN ('pending','queued','running','validating_files')
		ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*FineTuneJob
	for rows.Next() {
		j := &FineTuneJob{}
		if err := rows.Scan(&j.ID, &j.UserID, &j.Provider, &j.ProviderJobID, &j.Model,
			&j.FineTunedModel, &j.Status, &j.TrainingFile, &j.ValidationFile,
			&j.Hyperparameters, &j.ResultFiles, &j.TrainedTokens,
			&j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt, &j.FinishedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
