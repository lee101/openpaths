CREATE TABLE IF NOT EXISTS model_metadata (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        TEXT NOT NULL,
    model_id        TEXT NOT NULL,
    display_name    TEXT NOT NULL DEFAULT '',
    organization    TEXT NOT NULL DEFAULT '',
    model_type      TEXT NOT NULL DEFAULT 'chat',
    context_length  INT NOT NULL DEFAULT 0,
    parameters      TEXT NOT NULL DEFAULT '',
    license         TEXT NOT NULL DEFAULT '',
    link            TEXT NOT NULL DEFAULT '',
    input_price     NUMERIC(12,6) NOT NULL DEFAULT 0,
    output_price    NUMERIC(12,6) NOT NULL DEFAULT 0,
    price_per_image NUMERIC(12,6) NOT NULL DEFAULT 0,
    features        JSONB NOT NULL DEFAULT '[]',
    input_modalities  JSONB NOT NULL DEFAULT '["text"]',
    output_modalities JSONB NOT NULL DEFAULT '["text"]',
    deployment      JSONB NOT NULL DEFAULT '[]',
    raw_metadata    JSONB NOT NULL DEFAULT '{}',
    last_scraped_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, model_id)
);

CREATE INDEX IF NOT EXISTS idx_model_metadata_provider ON model_metadata (provider);
CREATE INDEX IF NOT EXISTS idx_model_metadata_type ON model_metadata (model_type);
