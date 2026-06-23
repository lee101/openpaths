-- Optional per-user response saving + private usage search.
-- When a user opts in, the inputs and outputs of their text (chat/messages) and
-- image generations are persisted here and made privately searchable via
-- pg_trgm (exact/substring) + a gobed semantic embedding (512-dim, stored as bytea).

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Per-user opt-in toggles.
ALTER TABLE users ADD COLUMN IF NOT EXISTS save_responses_text   BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS save_responses_images BOOLEAN NOT NULL DEFAULT FALSE;

-- Dogfood: enable both for the admin user.
UPDATE users
   SET save_responses_text = TRUE, save_responses_images = TRUE, updated_at = now()
 WHERE lower(email) = 'leepenkman@gmail.com';

CREATE TABLE IF NOT EXISTS saved_responses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id  TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL,                 -- 'text' | 'image'
    model       TEXT NOT NULL DEFAULT '',
    provider    TEXT NOT NULL DEFAULT '',
    prompt      TEXT NOT NULL DEFAULT '',      -- searchable input (last user msg / image prompt)
    input       TEXT NOT NULL DEFAULT '',      -- full input (messages transcript)
    output      TEXT NOT NULL DEFAULT '',      -- output text (chat content / revised prompt)
    image_url   TEXT NOT NULL DEFAULT '',      -- image kind: rehosted public URL
    thumb_url   TEXT NOT NULL DEFAULT '',
    width       INTEGER NOT NULL DEFAULT 0,
    height      INTEGER NOT NULL DEFAULT 0,
    tokens_in   INTEGER NOT NULL DEFAULT 0,
    tokens_out  INTEGER NOT NULL DEFAULT 0,
    cost_cents  BIGINT NOT NULL DEFAULT 0,
    embedding   BYTEA,                          -- 512 × float32 little-endian; NULL until embedded
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Browse / list (per user, per kind, newest first).
CREATE INDEX IF NOT EXISTS idx_saved_responses_user_kind ON saved_responses (user_id, kind, created_at DESC);
-- Exact / substring / fuzzy search over the input prompt and the output text.
CREATE INDEX IF NOT EXISTS idx_saved_responses_prompt_trgm ON saved_responses USING gin (prompt gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_saved_responses_output_trgm ON saved_responses USING gin (output gin_trgm_ops);
