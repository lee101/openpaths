CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    name            TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    disabled        BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- API Keys
CREATE TABLE IF NOT EXISTS api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash        TEXT NOT NULL UNIQUE,
    key_prefix      TEXT NOT NULL,
    name            TEXT NOT NULL DEFAULT 'Default',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at    TIMESTAMPTZ,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    rate_limit_rpm  INT NOT NULL DEFAULT 60
);

CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash) WHERE NOT revoked;
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys (user_id);

-- Credit Balances
CREATE TABLE IF NOT EXISTS credit_balances (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance_cents   BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT positive_balance CHECK (balance_cents >= 0)
);

-- Credit Transactions
CREATE TABLE IF NOT EXISTS credit_transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_cents    BIGINT NOT NULL,
    balance_after   BIGINT NOT NULL,
    tx_type         TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    reference_id    UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_credit_tx_user_id ON credit_transactions (user_id, created_at DESC);

-- Usage Logs
CREATE TABLE IF NOT EXISTS usage_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    api_key_id      UUID NOT NULL REFERENCES api_keys(id),
    model           TEXT NOT NULL,
    provider        TEXT NOT NULL,
    tokens_in       INT NOT NULL DEFAULT 0,
    tokens_out      INT NOT NULL DEFAULT 0,
    latency_ms      INT NOT NULL DEFAULT 0,
    tps             REAL NOT NULL DEFAULT 0,
    cost_cents      BIGINT NOT NULL DEFAULT 0,
    error           TEXT,
    status_code     INT NOT NULL DEFAULT 200,
    streaming       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_usage_logs_user_id ON usage_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_logs_model ON usage_logs (model, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_logs_provider ON usage_logs (provider, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs (created_at DESC);

-- Migrations tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INT PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
