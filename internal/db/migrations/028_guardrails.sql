-- Guardrails: named policies for spend, model/provider access, PII, prompt injection, custom regex.
-- Assignable to API keys or as a user-wide default. One assignment per target.

CREATE TABLE IF NOT EXISTS guardrails (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name              TEXT NOT NULL DEFAULT 'Untitled',
    limit_cents       BIGINT,
    reset_interval    TEXT,
    budget_actions    TEXT[] NOT NULL DEFAULT '{}',
    allowed_models    TEXT[] NOT NULL DEFAULT '{}',
    allowed_providers TEXT[] NOT NULL DEFAULT '{}',
    prompt_injection  JSONB NOT NULL DEFAULT '{}'::jsonb,
    sensitive_info    JSONB NOT NULL DEFAULT '{}'::jsonb,
    custom_filters    JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT guardrails_reset_interval_chk CHECK (
        reset_interval IS NULL OR reset_interval IN ('daily', 'weekly', 'monthly')
    )
);

CREATE INDEX IF NOT EXISTS idx_guardrails_user ON guardrails (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS guardrail_assignments (
    guardrail_id UUID NOT NULL REFERENCES guardrails(id) ON DELETE CASCADE,
    target_type  TEXT NOT NULL,
    target_id    TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (target_type, target_id),
    CONSTRAINT guardrail_assignments_type_chk CHECK (target_type IN ('api_key', 'user'))
);

CREATE INDEX IF NOT EXISTS idx_guardrail_assignments_policy ON guardrail_assignments (guardrail_id);

CREATE TABLE IF NOT EXISTS guardrail_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guardrail_id UUID REFERENCES guardrails(id) ON DELETE SET NULL,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id   UUID,
    stage        TEXT NOT NULL,
    action       TEXT NOT NULL,
    detail       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_guardrail_events_user ON guardrail_events (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_guardrail_events_budget_email
    ON guardrail_events (guardrail_id, stage, action, created_at DESC)
    WHERE stage = 'budget' AND action = 'email';
