# Playground v2 — conversion-focused upgrade

## Customer signal
- 141 total users on prod; 7 real (excl. e2e fixtures)
- 0 `stripe_topup` txns — zero paying customers
- 5 of 7 real users never made an API call — activation leak
- leepenkman drove 3,427 / 3,498 usage rows

## Goals (in priority order)
1. **Reduce friction for signed-in users with 0 balance** — they can't try anything unless we surface the top-up path inside the playground.
2. **Show the value** — OpenPaths is "one key, many models"; the compare UI already proves that. Make it shareable.
3. **Keep parity with backend catalog** — hardcoded 28-model list drifts. Fetch `/v1/models`.
4. **Bring the cost angle forward** — show $/1M input + output in the model picker and $$ estimate per turn. Builds routing-value narrative.

## Scope
1. Fetch `/v1/models` on mount and group by `owned_by`. Cache to localStorage (`op_models_cache`) with 1h TTL. Fallback to hardcoded list when fetch fails (offline, unauth).
2. Fetch `/account/balance` on mount when API key present. Display `$X.XXXX` in toolbar. When balance <= 0, replace the plain number with a red "Top up" button linking to `/account`.
3. Cost estimate per response using the model's `pricing` block. Compute: `(input_tokens * input_per_1M + output_tokens * output_per_1M) / 1_000_000`. Display as `$0.00123` under each assistant bubble. If pricing missing, omit.
4. Persist each pane's messages to localStorage keyed by `op_pg_pane_<modelId>`. Reload on mount. "Clear" wipes the stored convo too.
5. Share link button — copies `location.origin/playground?model=X&prompt=<last user msg>`. Query param `prompt` auto-runs on load if API key present.
6. Not-signed-in empty state — prominent "Sign in — it's free" button linking to `/account`, not just "Set API key".
7. Filter fetched models to chat-capable only (skip image/video/audio/embedding endpoints). Heuristic: exclude IDs with `image|video|audio|embed|tts|transcribe|whisper|flux|klein|wan|hailuo|kling|luma` unless they also support chat (keep simple — use provider/owned_by signals too).
8. Model selector dropdown shows `$in / $out per 1M` inline, plus capability icons (vision/tools).

## Out of scope (flag as follow-up)
- Image/Video/Audio tabs in playground. Big scope; separate plan.
- Tool calling UI. Separate.
- Saved prompt library. Separate.

## Verification
- Run `bun run build` for typecheck
- Manual smoke: `bun run dev`, open `/playground`, verify balance displays, models load from /v1, compare two models, refresh, verify persistence, click Share, paste elsewhere, verify prompt auto-fills.
- Deploy via `./deploy.sh site`. Purge Cloudflare.
