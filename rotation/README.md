# Provider key rotation

This directory contains scripts for provider APIs that can create a new API key programmatically and return the new secret.

Supported providers:

| Provider | Updates | Required credential |
|----------|---------|---------------------|
| OpenAI | `OPENAI_API_KEY` | `OPENAI_ADMIN_KEY` or `OPENAI_ADMIN_API_KEY`; `OPENAI_PROJECT_ID` when the admin key can access multiple active projects |
| fal | `FAL_API_KEY` | `FAL_ADMIN_API_KEY`, legacy `FAL_KEY`, or current `FAL_API_KEY` if it has admin key permissions |

OpenAI creates a project service account, validates the one-time returned API key against `/v1/models`, writes it to `.env`, and then revokes the previous `OPENAI_API_KEY` when it can uniquely identify it from the project API key list. If the previous key belongs to a service account, the script deletes the previous service account. fal creates an API key through the Platform API and uses the one-time returned full key.

Anthropic was checked, but its Admin API currently documents listing and updating API keys, not creating a new key secret through the API. Because a rotation script must be able to capture a newly returned secret, Anthropic is intentionally not included here.

## Usage

```bash
python3 rotation/rotate_provider_key.py openai
python3 rotation/rotate_provider_key.py fal
python3 rotation/rotate_provider_key.py all
```

The script reads credentials from the shell environment first and then from `.env`. It writes a timestamped `.env.bak-rotation-*` backup before replacing the target key.

`--no-env-write` creates and validates the provider key, but skips modifying `.env` and skips revoking the old key:

```bash
python3 rotation/rotate_provider_key.py openai --no-env-write
```
