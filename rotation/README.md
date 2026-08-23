# Provider key rotation

This directory contains a script that creates (or accepts) a replacement provider API key,
validates it, writes it into this repo's `.env` (with a timestamped backup), and then
best-effort revokes the previous key.

Two classes of provider:

- **Full rotation** — the provider exposes a key-management API that mints a new secret
  programmatically. We create, validate, install, and revoke the old key end to end.
- **BYO (bring your own)** — the provider only mints keys in its web console. Create the
  key there, export it as `*_NEW_API_KEY`, and the rotator validates + installs it. No revoke.

| Provider | Mode | Updates | Required credential |
|----------|------|---------|---------------------|
| OpenAI | full | `OPENAI_API_KEY` | `OPENAI_ADMIN_KEY`/`OPENAI_ADMIN_API_KEY`; `OPENAI_PROJECT_ID` when the admin key can access multiple active projects |
| fal | full | `FAL_API_KEY` | `FAL_ADMIN_API_KEY`, legacy `FAL_KEY`, or current `FAL_API_KEY` with admin permissions |
| OpenRouter | full | `OPENROUTER_API_KEY` | `OPENROUTER_PROVISIONING_KEY` (aliases: `OPENROUTER_PROVISIONING_API_KEY`, `OPENROUTER_MANAGEMENT_KEY`) |
| xAI | full | `XAI_API_KEY` | `XAI_MANAGEMENT_KEY` (aliases: `XAI_MANAGEMENT_API_KEY`, `XAI_ADMIN_KEY`); optional `XAI_TEAM_ID` (auto-discovered from the current `XAI_API_KEY` otherwise) |
| Anthropic | BYO | `ANTHROPIC_API_KEY` | `ANTHROPIC_NEW_API_KEY` or `ANTHROPIC_REPLACEMENT_API_KEY`; optional `ANTHROPIC_ADMIN_KEY`/`ANTHROPIC_ADMIN_API_KEY` to deactivate the previous key |
| Mistral | BYO | `MISTRAL_API_KEY` | `MISTRAL_NEW_API_KEY` or `MISTRAL_REPLACEMENT_API_KEY` |
| Nous | BYO | `NOUS_API_KEY` | `NOUS_NEW_API_KEY` or `NOUS_REPLACEMENT_API_KEY` |

**OpenAI** creates a project service account, validates the one-time returned key against `/v1/models`, writes it, then revokes the previous `OPENAI_API_KEY` (deleting its service account when applicable) once it uniquely identifies it. **fal** creates a key through the Platform API and uses the one-time returned full key.

**OpenRouter** uses the Provisioning API (`POST/GET/DELETE https://openrouter.ai/api/v1/keys`): it creates a key with the provisioning key, validates it via `GET /api/v1/key`, installs it, then deletes the previous key when it can uniquely match its redacted `label` to the old secret. The provisioning key is created in the OpenRouter dashboard and can only manage keys.

**xAI** uses the Management API (`https://management-api.x.ai`): it discovers the team via `GET https://api.x.ai/v1/api-key` (or `XAI_TEAM_ID`), creates a key under that team, validates it, installs it, then revokes the previous key deterministically by looking up its `api_key_id` (again via `/v1/api-key`) and issuing `DELETE /auth/api-keys/{id}`. The management key is separate from the inference key (Console → Settings → Management Keys).

**Anthropic / Mistral / Nous** only mint keys in their consoles. Create the new key there, export it as the provider's `*_NEW_API_KEY`, and the rotator validates it against the provider's `/v1/models` (Anthropic also deactivates the previous key when an admin key can uniquely identify it).

## Usage

```bash
python3 rotation/rotate_provider_key.py openai
python3 rotation/rotate_provider_key.py fal
python3 rotation/rotate_provider_key.py openrouter   # needs OPENROUTER_PROVISIONING_KEY
python3 rotation/rotate_provider_key.py xai          # needs XAI_MANAGEMENT_KEY
ANTHROPIC_NEW_API_KEY="sk-ant-api03-..." python3 rotation/rotate_provider_key.py anthropic
MISTRAL_NEW_API_KEY="..." python3 rotation/rotate_provider_key.py mistral
NOUS_NEW_API_KEY="..."    python3 rotation/rotate_provider_key.py nous
python3 rotation/rotate_provider_key.py all

# OpenRouter fan-out, including ~/.secretbashrc and OpenPaths prod. The purge
# is refused unless prod is updated and restarted first.
python3 rotation/rotate_openrouter_everywhere.py --deploy-prod --purge-others
```

Run the tests (no network — the HTTP layer is stubbed):

```bash
python3 -m unittest discover -s rotation -p 'test_*.py'
```

The script reads credentials from the shell environment first and then from `.env`. It writes a timestamped `.env.bak-rotation-*` backup before replacing the target key. If a target key appears more than once in `.env`, rotation collapses it to one updated entry.

`--no-env-write` creates and validates the provider key, but skips modifying `.env` and skips revoking the old key:

```bash
python3 rotation/rotate_provider_key.py openai --no-env-write
```

`rotate_openrouter_everywhere.py --deploy-prod` copies the updated repo `.env` to
OpenPaths prod and restarts only the `openpaths` supervisor service. It does not
touch the database, guardrail definitions, or assignments. `--purge-others`
requires `--deploy-prod` so old OpenRouter keys are not removed before prod has
the replacement.

Rotation-only management credentials can be moved out of `.env` into encrypted
`.envsecret.*` files with:

```bash
rotation/rotate-passwords.sh --init
rotation/rotate-passwords.sh
```

The command prompts for the password each time, decrypts only into a temporary
directory, and removes the plaintext files on exit. The password is never stored
in the repository or script.

Anthropic is supported when a freshly-created console key is supplied for one
run, for example: `ANTHROPIC_NEW_API_KEY='...' rotation/rotate-passwords.sh`.
Without that variable it is skipped safely; OpenAI and OpenRouter remain fully
automatable with the encrypted management bundles.
