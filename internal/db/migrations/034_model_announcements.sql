CREATE TABLE IF NOT EXISTS model_catalog_seen (
    model_id      TEXT PRIMARY KEY,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    announced_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS model_announcements (
    model_id   TEXT NOT NULL,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending',
    error      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (model_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_model_announcements_status ON model_announcements (model_id, status);
