CREATE TABLE IF NOT EXISTS user_provider_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    api_key     TEXT NOT NULL DEFAULT '',
    auth_json   TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_user_provider_keys_user ON user_provider_keys(user_id);
