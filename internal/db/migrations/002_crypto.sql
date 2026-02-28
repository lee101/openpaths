CREATE SEQUENCE IF NOT EXISTS crypto_deposit_index_seq;

CREATE TABLE IF NOT EXISTS crypto_checkout_intents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method          TEXT NOT NULL CHECK (method IN ('sol', 'codex')),
    amount_usd      NUMERIC(12,2) NOT NULL,
    amount_lamports BIGINT NOT NULL,
    amount_ui       TEXT NOT NULL,
    deposit_index   BIGINT NOT NULL,
    deposit_pubkey  TEXT NOT NULL,
    mint            TEXT,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'expired')),
    tx_sig          TEXT,
    credits_cents   BIGINT NOT NULL,
    swept           BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at      TIMESTAMPTZ NOT NULL,
    honor_until     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_crypto_checkout_status ON crypto_checkout_intents (status) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_crypto_checkout_user ON crypto_checkout_intents (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_crypto_checkout_swept ON crypto_checkout_intents (swept) WHERE status = 'paid' AND NOT swept;
