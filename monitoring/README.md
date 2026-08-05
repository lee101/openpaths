# monitoring/

Auto-fix monitoring for openpaths.io. Detects frontend errors from Playwright
route scans, browser-reported client errors, and backend supervisor errors. On a
12h debounce it spawns Codex through `monitoring/run_codex_agent.sh` to diagnose,
patch, redeploy, and verify.

## Layout

| File | Purpose |
| --- | --- |
| `error_watcher.sh` | **Main entrypoint.** Frontend + backend scan, debounced Codex spawn. |
| `check_js.mjs` | Headless Playwright scan of important routes; emits JSON. |
| `capture_frontend_client_errors.sh` | SSH + tails the backend JSONL file populated by real browsers. |
| `capture_backend_errors.sh` | SSH + tails supervisor log from a stored offset; finds new panics/SIGILL/etc. |
| `autofix_agent.sh` | Legacy 30-min site-up check + supervisor restart. Still wired. |
| `daily_route_monitor.sh` | Daily API route tests + Codex fixer. Still wired. |
| `monitor_site.py` | Long-running site watcher (espeak alerts). |
| `run_codex_agent.sh` | Shared Codex launcher: `$CODEX_LOCAL --dangerously-bypass-approvals-and-sandbox -m gpt-5.6-sol --config model_reasoning_effort=medium exec`. |
| `setup_monitoring.sh` | Installs all cron entries on the prod box. |
| `state/` | gitignored — offsets and `last_codex_fix_at`. |
| `errors/` | gitignored — JSON snapshots per detection. |
| `logs/` | gitignored — per-run logs. |

## Running

```bash
# Scan and (if needed, and if 12h elapsed) auto-fix
bash monitoring/error_watcher.sh

# Just check, never spawn Codex. Exit 0 = clean, 1 = errors found.
bash monitoring/error_watcher.sh --check-only

# Bypass the 12h debounce
bash monitoring/error_watcher.sh --force

# Detect-only, no Codex spawn
bash monitoring/error_watcher.sh --dry-run
```

Override knobs (env): `OPENPATHS_SITE`, `OPENPATHS_HOST`, `OPENPATHS_SSH_PASS`,
`DEBOUNCE_SECONDS` (default 43200 = 12h), `OPENPATHS_REMOTE_LOG`,
`OPENPATHS_FRONTEND_REMOTE_LOG`, `CODEX_LOCAL`, `CODEX_MODEL`,
`CODEX_REASONING_EFFORT`.

## Debounce

`monitoring/state/last_codex_fix_at` stores the unix epoch of the last Codex
spawn. While `now - last < DEBOUNCE_SECONDS`, the watcher keeps logging errors
to `monitoring/errors/` but skips the Codex spawn. This protects against
flapping outages turning into 24 parallel fix agents.

## Frontend Client Errors

The React app installs a tiny reporter in `src/lib/frontendErrors.ts`. Browser
`error`, `unhandledrejection`, and `console.error` events are posted to
`POST /monitoring/frontend-errors`, and the Go server appends them to
`monitoring/errors/frontend_client.jsonl` on the backend. The watcher tails that
file by byte offset and inlines fresh events into the Codex prompt.

## What gets fed to Codex

Every detected error JSON file (frontend + backend) is inlined into the prompt
along with SSH details, build commands, and a strict directive to run
`monitoring/error_watcher.sh --check-only` at the end to verify the fix took.
