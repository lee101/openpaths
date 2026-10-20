# Nous MissingNo Rollout

Local implementation is staged, not a verified production launch. Do not send
the announcement until the authenticated checks and artifact review succeed.

## Catalog

The 2026-10-11 `https://inference-api.nousresearch.com/v1/models` response lists
`stealth/missingno` with zero prompt/completion prices, 262144 context tokens,
131072 maximum completion tokens, tools, text-only input, and mandatory reasoning.
`openpaths/free` aliases the OpenPaths `missingno` model, without paid fallbacks.
Other Nous routes use separate `nous/` IDs so direct DeepSeek and Z.ai models
keep their existing routes. No Google route is added.

Explicit `:free` variants and paid variants are not interchangeable. Cached
input is a separate API price, not a discount to ordinary uncached input.
The requested DeepSeek `0423` IDs are absent from the live catalog and omitted.

## Pricing

Ordinary chat responses and stream chunks include `routing` with the actual
served model, provider, USD input/cache/output rates per million tokens. The
outer `model` remains the client's requested ID for compatibility. Existing
chat billing deducts against the actual candidate model, including fallbacks.
Rates are catalog snapshots, not a guarantee against future upstream price
changes. Recheck rates before deployment; never introduce a paid fallback into
`openpaths/free`. Early stream rates use the base tier until usage is available.

## Launch Gates

1. Provision `NOUS_API_KEY` securely on production. It was missing locally and
   in production `.env` at inspection. Unauthenticated chat uses x402 payments;
   MissingNo is unavailable through that payment path.
2. Run `NOUS_API_KEY=... python3 scripts/benchmark-nous-missingno.py` with the key
   supplied securely by the process environment, not a logged shell command.
   The script saves the request, raw response, elapsed time, token usage and SVG.
   The house prompt uses mandatory `medium` reasoning, unlike previous
   thinking-disabled runs; disclose that difference. No retries are automatic.
3. Render and review the SVG. Check bird anatomy, frame, wheels and animation.
   Record limitations honestly. A single SVG sample is not a coding benchmark
   score and does not establish the anonymous model's identity.
4. Publish the reviewed sample as
   `public/static/blog/pelican-svg/missingno.svg`, plus a write-up with measured
   usage/timing and visual findings. Add those findings to the email draft.
5. Apply only this task's changes to the dirty production worktree. Build,
   restart, then test authenticated OpenPaths `openpaths/free` non-streaming,
   streaming and tool calls. Verify zero charge and `routing.model=missingno`.
6. Use the existing announcement service with `model_id=missingno`,
   `template=announce-free-missingno.html`: dry-run first, then a test email,
   then confirmed send. Preserve suppression/unsubscribe/disabled-account
   filtering and deduplication. Monitor sent/failed counts to completion.

Do not send the staged email while its benchmark asset is absent, before the
performance findings are added, or while the production route is unverified.
