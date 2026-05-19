CREATE TABLE IF NOT EXISTS video_generation_jobs (
    id              TEXT PRIMARY KEY,
    request_hash    TEXT NOT NULL,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id      UUID,
    model           TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued',
    request_json    JSONB NOT NULL,
    result_json     JSONB,
    error_type      TEXT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS video_generation_jobs_active_hash_idx
    ON video_generation_jobs (request_hash)
    WHERE status IN ('queued', 'running', 'completed');

CREATE INDEX IF NOT EXISTS video_generation_jobs_user_created_idx
    ON video_generation_jobs (user_id, created_at DESC);
