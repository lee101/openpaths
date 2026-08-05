CREATE TABLE IF NOT EXISTS research_jobs (
    id            TEXT PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id    UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    query         TEXT NOT NULL,
    depth         TEXT NOT NULL DEFAULT 'standard',
    model         TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'queued',
    steps         JSONB NOT NULL DEFAULT '[]'::jsonb,
    report        TEXT NOT NULL DEFAULT '',
    citations     JSONB NOT NULL DEFAULT '[]'::jsonb,
    cost_units    BIGINT NOT NULL DEFAULT 0,
    error_type    TEXT,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS research_jobs_user_created_idx ON research_jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS research_jobs_status_idx ON research_jobs (status);
