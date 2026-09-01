# Provider credit polling

`polling.ProviderCreditMonitor` makes one tiny direct request to a curated
low-cost chat model for each configured cloud provider. It deliberately calls
the provider registry directly rather than `/v1/chat/completions`, so model
fallbacks cannot conceal an exhausted upstream account.

The monitor runs every day at **08:00 Pacific/Auckland** (NZST/NZDT is handled
by the IANA timezone database) with the prompt `say hi nothing else` and a
16-token output cap. It sends one combined operational email only when an
upstream response explicitly indicates exhausted credits, balance, billing,
spending limits, payment, or account quota. Authentication errors, ordinary
RPM rate limits, timeouts, and provider outages are logged but do not email.

Defaults:

- recipient: `leepenkman@gmail.com`
- time: `08:00`
- timezone: `Pacific/Auckland`
- per-provider timeout: `90s`

Environment overrides:

- `OPENPATHS_CREDIT_POLL_DISABLED=1`
- `OPENPATHS_CREDIT_POLL_EMAIL=address@example.com`
- `OPENPATHS_CREDIT_POLL_TIME=08:00`
- `OPENPATHS_CREDIT_POLL_TIMEZONE=Pacific/Auckland`
- `OPENPATHS_CREDIT_POLL_TIMEOUT=90s`

Email delivery uses the existing AWS SES SMTP configuration. The provider list
and chosen models live in `targetSpecs` in `provider_credits.go`; keep these on
paid but inexpensive routes, because a free route can still succeed when paid
credits are depleted.

Current direct targets:

| Provider | Model |
| --- | --- |
| xAI | `grok-4.6` |
| OpenAI | `gpt-5.4-nano` |
| Google | `gemini-3.1-flash-lite` |
| Anthropic | `claude-haiku-4-5-20251001` |
| DeepSeek | `deepseek-chat` |
| Mistral | `open-mistral-nemo` |
| Groq | `llama-3.1-8b-instant` |
| OpenRouter | `openpaths/chat-latest` |
| Together | `together/deepseek-v3.1` |
| MiniMax | `minimax-m2` |
| Z.AI | `glm-4.6v-flashx` |
| Nous | `hermes-4-70b` |
| Fireworks | `fireworks/gpt-oss-120b` |
| NVIDIA | `nvidia/deepseek-v3.2` |
| Cursor | `composer-2.5` |
