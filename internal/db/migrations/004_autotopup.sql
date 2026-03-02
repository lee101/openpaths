-- Auto-topup + Stripe columns on users
ALTER TABLE users ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS stripe_payment_method_id TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS autotopup_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS autotopup_threshold_cents BIGINT NOT NULL DEFAULT 50000;
ALTER TABLE users ADD COLUMN IF NOT EXISTS autotopup_amount_cents BIGINT NOT NULL DEFAULT 100000;
ALTER TABLE users ADD COLUMN IF NOT EXISTS autotopup_last_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS autotopup_charges (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_cents            BIGINT NOT NULL,
    amount_usd              NUMERIC(12,2) NOT NULL,
    stripe_payment_intent_id TEXT,
    status                  TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'succeeded', 'failed')),
    error                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_autotopup_charges_user_id ON autotopup_charges (user_id, created_at DESC);
