# provisioning/

Automated Google Cloud setup for OpenPaths. Currently: Gemini API key provisioning.

OpenPaths calls Gemini through the **Gemini Developer API**
(`generativelanguage.googleapis.com`) using an **API key**, read from the
`GEMINI_API_KEY` environment variable (`internal/config/config.go` →
`google` provider in `config.yaml`). It does **not** use Vertex AI or ADC for
this path, so a scoped API key is all that's needed.

## Prerequisites

```bash
gcloud auth login                       # authenticate a user with project access
gcloud config set project openpaths-498620
```

## Provision a Gemini key

```bash
# create (or reuse) a key restricted to generativelanguage.googleapis.com, then smoke-test it
./provisioning/provision-gemini.sh

# same, and write it into .env (GEMINI_API_KEY), backing up .env first
./provisioning/provision-gemini.sh --write
```

The script is **idempotent**: it reuses the key named `openpaths-gemini`
(override with `KEY_DISPLAY_NAME=`) instead of creating duplicates. It:

1. enables `apikeys.googleapis.com` + `generativelanguage.googleapis.com`,
2. creates/reuses an API key restricted to the Gemini API,
3. fetches the key string,
4. smoke-tests `generateContent` (retrying through new-key propagation),
5. optionally writes `GEMINI_API_KEY` into `.env`.

### Test exit states

| Result | Meaning | Action |
|--------|---------|--------|
| `HTTP 200` | key fully working | none |
| `HTTP 429 prepaid credits depleted` | key + project valid, billing out of credits | top up at <https://ai.studio/projects> → billing |
| `HTTP 400 API Key not found` (transient) | new key still propagating | re-run in a few minutes |
| `HTTP 403 project denied` | project suspended or key misrestricted | use a different project / contact support |

## ADC (only if needed for non-Gemini Google libraries)

`./provisioning/fix-adc.sh` diagnoses a stale `GOOGLE_APPLICATION_CREDENTIALS`
and prints the commands to re-establish user ADC. Not required for OpenPaths'
Gemini path.

## Manual equivalents

```bash
# create restricted key
gcloud services api-keys create --display-name=openpaths-gemini \
  --api-target=service=generativelanguage.googleapis.com

# fetch the string
gcloud services api-keys get-key-string <KEY_RESOURCE_NAME> --format='value(keyString)'

# test
curl -s "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=$GEMINI_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"contents":[{"parts":[{"text":"Reply OK"}]}]}'
```
