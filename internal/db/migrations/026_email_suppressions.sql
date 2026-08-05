CREATE TABLE IF NOT EXISTS email_suppressions (
    email       TEXT PRIMARY KEY,
    reason      TEXT NOT NULL DEFAULT 'unsubscribe',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO email_suppressions (email, reason)
VALUES ('shopping@oreshkin.org', 'manual_block')
ON CONFLICT (email) DO NOTHING;
