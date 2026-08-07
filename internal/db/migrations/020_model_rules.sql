-- Model IAM: opt-in, open-by-default allow/deny rules over model ids.
-- No rows for a user (or their teams) => every model is allowed. Rules attach
-- hierarchically (user + team); evaluation is most-specific-wins with
-- deny-overrides at equal specificity.

CREATE TABLE IF NOT EXISTS model_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       TEXT NOT NULL,          -- 'user' | 'team'
    scope_id    TEXT NOT NULL,          -- user id or team id
    model_glob  TEXT NOT NULL,          -- '*', 'anthropic/*', 'gpt-5-codex'
    effect      TEXT NOT NULL,          -- 'allow' | 'deny'
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_model_rules_scope ON model_rules (scope_id);
