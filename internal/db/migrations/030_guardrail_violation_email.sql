-- Optional email notifications for model/provider access violations.
ALTER TABLE guardrails
    ADD COLUMN IF NOT EXISTS email_on_violation BOOLEAN NOT NULL DEFAULT FALSE;
