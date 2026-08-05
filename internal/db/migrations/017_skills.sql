-- Searchable agent-skill library: self-contained markdown skills ingested from
-- the skill repos (codex-infinity, hermes-agent). Backed by pg_trgm exact/substring
-- search + a gobed semantic index (internal/skillindex). Populated by cmd/skill-ingest.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS skills (
    id              TEXT PRIMARY KEY,
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL DEFAULT '',
    source          TEXT NOT NULL DEFAULT '',
    source_repo     TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL DEFAULT '',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    setup_preamble  TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_skills_name_trgm ON skills USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_skills_desc_trgm ON skills USING gin (description gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_skills_tags ON skills USING gin (tags);
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills (category);
CREATE INDEX IF NOT EXISTS idx_skills_source ON skills (source);
