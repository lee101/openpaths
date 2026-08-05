-- Billshock guards: opt-in billing alerts + a max top-up cap. All default to 0
-- (disabled / unlimited) so existing accounts are unaffected. Kept off the
-- userCols scan list; read directly via GuardQueries.

ALTER TABLE users ADD COLUMN IF NOT EXISTS billing_alert_threshold_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS billing_alert_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS billing_alert_last_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS max_topup_cap_cents BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS team_billing (
    team_id              TEXT PRIMARY KEY,
    alert_threshold_cents BIGINT NOT NULL DEFAULT 0,
    max_topup_cap_cents   BIGINT NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
