# Deploy Wave — GLM 5.1, auto-topup, drip emails, Fal

Scope: ship everything that is currently WIP in the tree, plus a little extra,
in one deploy to openpaths.io. Target: site + api both out tonight.

## Status of in-flight work (pre-existing diffs)

- `config.yaml` — new `glm-5.1` (together + zai fallback), new `minimax-m2.7`,
  transcription models (whisper/groq/fireworks/openai), auto tier rewrite
  (easy→gemini-flash-lite, medium→claude-sonnet), removed fictional gpt-5.4
  entries and `or/hunter-alpha`. Port moved 8090 → 8092. Renamed
  `openpath-embed` → `openpaths-embed`.
- `src/data/models.ts` — GLM-5.1 & MiniMax-M2.7 entries.
- `src/pages/Account.tsx` — autotopup card, embedded Stripe payment-method
  setup (Elements), low-balance banner, overview autotopup card,
  recommended-topup buttons, saved-card management.
- `internal/handler/auth.go` — OnRegister callback plumbed in for drip welcome.
- `internal/handler/account_stats.go` — new per-user stats endpoints
  (timeseries, spend by key, spend by provider + drilldowns).
- `internal/db/queries/stats.go` — per-user versions of all stats queries.
- `internal/db/migrations/007_autotopup_default_refresh.sql` — refresh
  autotopup defaults to $100 threshold / $200 amount.
- `internal/email/{drip.go,sender.go}` — drip runner + SES SMTP sender.
- `emails/*.html` + `drip_config.json` — 20-email onboarding drip.
- `internal/provider/zai/zai_integration_test.go` — GLM-5.1 integration test.
- `cmd/openpaths/main.go` — wires drip runner into startup.
- `internal/server/server.go` — wires drip OnRegister + per-user stats routes.
- `e2e/account.spec.ts` — updated for $100 default topup, new autotopup card,
  payment-method setup / save / delete, balance-card replacing Overview h1.
- `scripts/gen_og_*.py` + `public/og-image.png` — new branding OG images.
- `cmd/discover/main.go` — new provider-discovery probe CLI.

## New work for this wave

### 1. Wire GLM-5.1 fallback properly
- `config.yaml`: `glm-5.1` currently has `fallback_providers: ["zai"]`, but
  the fallback reuses `provider_model_id: zai-org/GLM-5.1` which zai will
  reject. Mirror the existing `zai/glm-5` pattern and add a `zai/glm-5.1`
  model, then have `glm-5.1` fall back via `fallback_models: ["zai/glm-5.1"]`.

### 2. Fal provider + fal email
- Add a minimal `fal` provider module that can serve image-gen (flux-schnell,
  flux-dev, flux-1.1-pro) and video-gen (kling-v1.5, luma-dream-machine) so
  users can route `/v1/images/generations` + `/v1/videos/generations` to fal.
- Register fal models in `config.yaml`.
- Add one fal spotlight email to drip chain (between qwen and inference-
  providers) — 30 days out, e.g. "Fal: The Fastest Path to Flux & Video".

### 3. Opus 4.7
- Add `claude-opus-4-7` to `config.yaml` as the latest anthropic model.
- Point the `opus-latest` alias at 4.7 (was 4.6 or 4.1 previously).
- Mirror in `src/data/models.ts`.
- Check any hard-coded references to older opus IDs and update.

### 4. test-mode Stripe e2e
- Existing `e2e/checkout-flow.spec.ts` registers, logs in, opens stripe and
  verifies the embedded checkout iframe attaches. The user wants us to take
  it one step further: a `?test=true` (or `stripe_test=true`) query param that
  forces the `STRIPE_PUBLISHABLE_KEY_TEST` / `STRIPE_SECRET_KEY_TEST` on the
  backend, then fill in Stripe test card `4242 4242 4242 4242`, submit, and
  verify balance increments.
- Because Stripe's embedded Checkout iframe is same-origin-restricted, the
  cleanest way is to fill the test card via Playwright's iframe locator.
- Update `e2e/checkout-flow.spec.ts` + `account.spec.ts` to use the same
  amount defaults as the UI (25/100/200/500) — they still reference $10
  which was removed.

### 5. Clean up email templates
- `emails/` has duplicate older templates (`04-flagship-models.html`,
  `07-art-generation.html`, `08-reasoning-models.html`, etc.) from an older
  campaign not in `drip_config.json`. Decide: either delete, or repurpose
  `07-art-generation.html` + `11-video-generation.html` as the fal email.
  Prefer the latter — less churn, matches the user's "add fal to email chain".

### 6. Ship
- `go build ./...`, `go test ./...` for core
- `npm run build` locally
- Commit in logical groupings (see below)
- `./deploy.sh env` then `./deploy.sh api` then `./deploy.sh site`
- Smoke test openpaths.io: login, add $10 via stripe test card, confirm
  balance updates, check `/v1/chat/completions` with `glm-5.1` model.

## Commit plan (rough groupings)
1. models + providers: glm-5.1 wiring, minimax-m2.7, transcription, fal
2. auto-topup + stripe embedded setup: migration + Account.tsx + e2e
3. per-user stats endpoints
4. email drip infra + 20 templates
5. misc: env.example SES vars, OG images, discover cmd

Keep the commits clean so the OSS repo stays readable.
