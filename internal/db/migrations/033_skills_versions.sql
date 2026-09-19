-- Versioned skills marketplace: owner-only writes, immutable versions, file attachments.
-- Next free number after 031_skills.sql (032 is taken by eval_results).

ALTER TABLE skills ADD COLUMN IF NOT EXISTS owner_id TEXT NOT NULL DEFAULT '';
ALTER TABLE skills ADD COLUMN IF NOT EXISTS current_version TEXT NOT NULL DEFAULT '1.0.0';

CREATE TABLE IF NOT EXISTS skill_versions (
    skill_id     TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version      TEXT NOT NULL,
    body         TEXT NOT NULL DEFAULT '',
    setup_script TEXT NOT NULL DEFAULT '',
    setup_prompt TEXT NOT NULL DEFAULT '',
    skill_prompt TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (skill_id, version)
);

CREATE TABLE IF NOT EXISTS skill_files (
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version  TEXT NOT NULL,
    path     TEXT NOT NULL,
    mime     TEXT NOT NULL DEFAULT '',
    size     INT NOT NULL DEFAULT 0,
    content  BYTEA,
    PRIMARY KEY (skill_id, version, path)
);

CREATE INDEX IF NOT EXISTS idx_skill_versions_skill ON skill_versions (skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_files_skill_version ON skill_files (skill_id, version);
