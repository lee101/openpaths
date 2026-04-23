-- Track cumulative refunded amount per Stripe checkout session.
-- Source of truth for idempotency on charge.refunded events: each event carries
-- Stripe's cumulative amount_refunded; we claw back only the delta vs what
-- we've already deducted.
ALTER TABLE stripe_deposits
    ADD COLUMN IF NOT EXISTS refunded_cents BIGINT NOT NULL DEFAULT 0;
