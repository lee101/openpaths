CREATE TABLE IF NOT EXISTS eval_results (
    suite             TEXT        NOT NULL,
    case_id           TEXT        NOT NULL,
    model             TEXT        NOT NULL,
    passed            BOOLEAN     NOT NULL DEFAULT FALSE,
    score             REAL        NOT NULL DEFAULT 0,
    ttft_ms           INT         NOT NULL DEFAULT 0,
    total_ms          INT         NOT NULL DEFAULT 0,
    prompt_tokens     INT         NOT NULL DEFAULT 0,
    completion_tokens INT         NOT NULL DEFAULT 0,
    tokens_per_sec    REAL        NOT NULL DEFAULT 0,
    cost_micro_usd    BIGINT      NOT NULL DEFAULT 0,
    answer_preview    TEXT        NOT NULL DEFAULT '',
    error             TEXT,
    ran_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (suite, case_id, model)
);

CREATE INDEX IF NOT EXISTS idx_eval_results_ran_at ON eval_results (ran_at DESC);
CREATE INDEX IF NOT EXISTS idx_eval_results_suite_score ON eval_results (suite, score DESC);
