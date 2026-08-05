-- Share-a-chat: public read-only chat transcripts served at /chat/:slug.

CREATE TABLE IF NOT EXISTS shared_chats (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL DEFAULT '',
    messages      JSONB NOT NULL,
    user_id       TEXT,
    views         INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_shared_chats_user ON shared_chats (user_id, created_at DESC);
