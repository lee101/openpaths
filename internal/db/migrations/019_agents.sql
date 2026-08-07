-- User-built agents: tool-use + computer-use agents with connected data sources.
-- Data sources (documents / databases / urls) are converted to markdown, chunked,
-- and embedded with gobed (512-dim float32 little-endian bytea) for in-process
-- semantic RAG, with a pg_trgm fallback over chunk content.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS agents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug          TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT 'auto',
    config        JSONB NOT NULL DEFAULT '{}'::jsonb,  -- {tools:[],max_steps,temperature}
    preset        TEXT NOT NULL DEFAULT '',            -- preset key this was created from
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agents_user ON agents (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_data_sources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id    UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,                          -- 'document' | 'database' | 'url'
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'ready',          -- 'ready' | 'ingesting' | 'error'
    meta        JSONB NOT NULL DEFAULT '{}'::jsonb,     -- {dsn,driver} for database; {url} for url
    chunk_count INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_sources_agent ON agent_data_sources (agent_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_documents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_source_id UUID NOT NULL REFERENCES agent_data_sources(id) ON DELETE CASCADE,
    agent_id       UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title          TEXT NOT NULL DEFAULT '',
    chunk_index    INTEGER NOT NULL DEFAULT 0,
    content        TEXT NOT NULL DEFAULT '',
    embedding      BYTEA,                                -- 512 × float32 little-endian; NULL until embedded
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_docs_agent ON agent_documents (agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_docs_source ON agent_documents (data_source_id);
CREATE INDEX IF NOT EXISTS idx_agent_docs_content_trgm ON agent_documents USING gin (content gin_trgm_ops);

CREATE TABLE IF NOT EXISTS agent_runs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id    UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    input       TEXT NOT NULL DEFAULT '',
    output      TEXT NOT NULL DEFAULT '',
    steps       JSONB NOT NULL DEFAULT '[]'::jsonb,
    status      TEXT NOT NULL DEFAULT 'completed',      -- 'completed' | 'error'
    cost_cents  BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_agent ON agent_runs (agent_id, created_at DESC);
