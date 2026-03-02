CREATE TABLE IF NOT EXISTS fine_tuning_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    provider_job_id TEXT NOT NULL DEFAULT '',
    model           TEXT NOT NULL,
    fine_tuned_model TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    training_file   TEXT NOT NULL,
    validation_file TEXT,
    hyperparameters JSONB NOT NULL DEFAULT '{}',
    result_files    JSONB NOT NULL DEFAULT '[]',
    trained_tokens  BIGINT NOT NULL DEFAULT 0,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ft_jobs_user_id ON fine_tuning_jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ft_jobs_status ON fine_tuning_jobs (status);
CREATE INDEX IF NOT EXISTS idx_ft_jobs_provider ON fine_tuning_jobs (provider, provider_job_id);

CREATE TABLE IF NOT EXISTS fine_tuning_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES fine_tuning_jobs(id) ON DELETE CASCADE,
    type            TEXT NOT NULL DEFAULT 'message',
    level           TEXT NOT NULL DEFAULT 'info',
    message         TEXT NOT NULL DEFAULT '',
    data            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ft_events_job_id ON fine_tuning_events (job_id, created_at);

CREATE TABLE IF NOT EXISTS fine_tuning_files (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL DEFAULT '',
    provider_file_id TEXT NOT NULL DEFAULT '',
    filename        TEXT NOT NULL,
    purpose         TEXT NOT NULL DEFAULT 'fine-tune',
    bytes           BIGINT NOT NULL DEFAULT 0,
    storage_url     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ft_files_user_id ON fine_tuning_files (user_id);
