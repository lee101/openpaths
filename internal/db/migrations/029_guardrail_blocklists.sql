-- Explicit deny rules for guardrails. Deny rules win over allow rules.
ALTER TABLE guardrails
    ADD COLUMN IF NOT EXISTS blocked_models TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS blocked_providers TEXT[] NOT NULL DEFAULT '{}';
