-- Image prompt search engine: our own DB of record for generated art images.
-- Imports all cutedsl images (reused in place from cutedsl.cc/images) plus newly
-- generated portrait/landscape variants stored on openpathsstatic.openpaths.io.
-- Backed by pg_trgm exact/substring search + a gobed (GPU) semantic index.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS art_images (
    id          TEXT PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    prompt      TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    width       INTEGER NOT NULL DEFAULT 1024,
    height      INTEGER NOT NULL DEFAULT 1024,
    aspect      TEXT NOT NULL DEFAULT 'square', -- 'square' | 'portrait' | 'wide'
    image_url   TEXT NOT NULL DEFAULT '',
    thumb_url   TEXT NOT NULL DEFAULT '',
    model       TEXT NOT NULL DEFAULT 'zimage',
    seed        BIGINT NOT NULL DEFAULT 0,
    steps       INTEGER NOT NULL DEFAULT 0,
    source      TEXT NOT NULL DEFAULT 'cutedsl', -- 'cutedsl' | 'openpaths-gen'
    tags        TEXT[] NOT NULL DEFAULT '{}',
    nsfw        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Exact / substring / fuzzy prompt search.
CREATE INDEX IF NOT EXISTS idx_art_images_prompt_trgm ON art_images USING gin (prompt gin_trgm_ops);
-- Subtag facets + tag pages.
CREATE INDEX IF NOT EXISTS idx_art_images_tags ON art_images USING gin (tags);
-- Browse / filter.
CREATE INDEX IF NOT EXISTS idx_art_images_aspect ON art_images (aspect, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_art_images_created ON art_images (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_art_images_source ON art_images (source);
-- Dimension search.
CREATE INDEX IF NOT EXISTS idx_art_images_dims ON art_images (width, height);
