CREATE TABLE IF NOT EXISTS artifacts (
    id           TEXT PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug         TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL DEFAULT 'Untitled',
    description  TEXT NOT NULL DEFAULT '',
    image_url    TEXT NOT NULL DEFAULT '',
    files        JSONB NOT NULL DEFAULT '[]'::jsonb,
    entry        TEXT NOT NULL DEFAULT 'index.html',
    visibility   TEXT NOT NULL DEFAULT 'private',
    tags         TEXT[] NOT NULL DEFAULT '{}',
    fork_of      TEXT,
    view_count   BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS artifacts_user_updated_idx ON artifacts (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS artifacts_public_published_idx ON artifacts (visibility, published_at DESC);
CREATE INDEX IF NOT EXISTS artifacts_tags_idx ON artifacts USING gin (tags);
CREATE INDEX IF NOT EXISTS artifacts_title_trgm_idx ON artifacts USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS artifacts_desc_trgm_idx ON artifacts USING gin (description gin_trgm_ops);
