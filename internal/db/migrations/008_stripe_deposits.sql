-- Track which Stripe checkout sessions have been credited.
-- Used as the idempotency boundary for both the webhook and the
-- reconciler-on-balance-fetch fallback.
CREATE TABLE IF NOT EXISTS stripe_deposits (
    session_id         TEXT PRIMARY KEY,
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_intent_id  TEXT,
    amount_total_cents BIGINT NOT NULL,
    credit_tx_id       UUID REFERENCES credit_transactions(id),
    source             TEXT NOT NULL,
    credited_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stripe_deposits_user_id
    ON stripe_deposits (user_id, credited_at DESC);

-- Backfill Paul's manual credit so the reconciler doesn't double-credit him.
INSERT INTO stripe_deposits
    (session_id, user_id, payment_intent_id, amount_total_cents, credit_tx_id, source, credited_at)
SELECT
    'cs_live_a17ciEKbwIsgoQIYP52GFqAa6JUe9VcquCfyAXUHbUzWDPqWFFBjnlNNni',
    u.id,
    'pi_3TLhDiHXPxordcGk1005ZVvS',
    2500,
    ct.id,
    'manual',
    now()
FROM users u
JOIN credit_transactions ct
  ON ct.id = '1ac89341-289f-40f9-99e9-f852640e0211'
 AND ct.user_id = u.id
WHERE u.id = '4fcbcb6d-eaf0-49e7-a2b2-80cb42cbdb25'
ON CONFLICT (session_id) DO NOTHING;
