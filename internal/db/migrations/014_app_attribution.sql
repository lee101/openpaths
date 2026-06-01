CREATE TABLE IF NOT EXISTS ai_apps (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                TEXT NOT NULL UNIQUE,
    name                TEXT NOT NULL,
    url                 TEXT NOT NULL DEFAULT '',
    description         TEXT NOT NULL DEFAULT '',
    favicon_url         TEXT NOT NULL DEFAULT '',
    categories          JSONB NOT NULL DEFAULT '[]',
    source              TEXT NOT NULL DEFAULT 'openpaths',
    external_path       TEXT NOT NULL DEFAULT '',
    total_tokens        BIGINT NOT NULL DEFAULT 0,
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_apps_url_nonempty ON ai_apps (url) WHERE url <> '';
CREATE INDEX IF NOT EXISTS idx_ai_apps_source ON ai_apps (source);
CREATE INDEX IF NOT EXISTS idx_ai_apps_total_tokens ON ai_apps (total_tokens DESC);

ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS app_id UUID REFERENCES ai_apps(id);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS app_url TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS app_title TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS app_categories JSONB NOT NULL DEFAULT '[]';

CREATE INDEX IF NOT EXISTS idx_usage_logs_app_id ON usage_logs (app_id, created_at DESC) WHERE app_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS app_daily_usage (
    day                 DATE NOT NULL,
    app_id              UUID NOT NULL REFERENCES ai_apps(id) ON DELETE CASCADE,
    source              TEXT NOT NULL DEFAULT 'openpaths',
    model               TEXT NOT NULL DEFAULT '',
    provider            TEXT NOT NULL DEFAULT '',
    requests            BIGINT NOT NULL DEFAULT 0,
    tokens_in           BIGINT NOT NULL DEFAULT 0,
    tokens_out          BIGINT NOT NULL DEFAULT 0,
    total_tokens        BIGINT NOT NULL DEFAULT 0,
    cost_cents          BIGINT NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (day, app_id, source, model, provider)
);

CREATE INDEX IF NOT EXISTS idx_app_daily_usage_app_day ON app_daily_usage (app_id, day DESC);
CREATE INDEX IF NOT EXISTS idx_app_daily_usage_tokens ON app_daily_usage (total_tokens DESC);
