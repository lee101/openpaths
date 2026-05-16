CREATE TABLE IF NOT EXISTS credential_failure_notifications (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL,
    credential_hash TEXT NOT NULL,
    last_sent_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, provider, credential_hash)
);
