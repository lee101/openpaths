CREATE TABLE IF NOT EXISTS model_probe_results (
    model              TEXT PRIMARY KEY,
    provider           TEXT NOT NULL DEFAULT '',
    latency_ms         INT NOT NULL DEFAULT 0,
    ok                 BOOLEAN NOT NULL DEFAULT FALSE,
    status_code        INT NOT NULL DEFAULT 0,
    error              TEXT,
    response_preview   TEXT NOT NULL DEFAULT '',
    probed_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_model_probe_results_probed_at ON model_probe_results (probed_at DESC);
CREATE INDEX IF NOT EXISTS idx_model_probe_results_ok ON model_probe_results (ok, probed_at DESC);
