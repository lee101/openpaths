-- Users who never saved alert settings get the default $5 low-balance alert (if they have deposited).
ALTER TABLE users ADD COLUMN IF NOT EXISTS billing_alert_configured BOOLEAN NOT NULL DEFAULT false;
UPDATE users SET billing_alert_configured = true WHERE billing_alert_enabled OR billing_alert_threshold_cents > 0;
