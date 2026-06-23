#!/usr/bin/env bash
# Provision a Gemini Developer API key (generativelanguage.googleapis.com) for OpenPaths.
#
# Idempotent: re-running reuses the existing key with the same display name instead
# of creating duplicates. Requires an authenticated gcloud (`gcloud auth login`) with
# rights on the target project.
#
# Usage:
#   ./provisioning/provision-gemini.sh                 # provision + test, print key
#   ./provisioning/provision-gemini.sh --write         # also write GEMINI_API_KEY into .env (backs up first)
#   PROJECT=my-proj ./provisioning/provision-gemini.sh # override project
#
# The OpenPaths server reads the key from GEMINI_API_KEY (see internal/config/config.go),
# which feeds the `google` provider in config.yaml.
set -euo pipefail

PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
KEY_DISPLAY_NAME="${KEY_DISPLAY_NAME:-openpaths-gemini}"
GL_SERVICE="generativelanguage.googleapis.com"
TEST_MODEL="${TEST_MODEL:-gemini-2.5-flash}"
WRITE_ENV=0
[[ "${1:-}" == "--write" ]] && WRITE_ENV=1

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$REPO_ROOT/.env"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarn:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

command -v gcloud >/dev/null || die "gcloud not found on PATH"
[[ -n "$PROJECT" ]] || die "no project set; pass PROJECT=... or run 'gcloud config set project <id>'"

# --- preflight -------------------------------------------------------------
ACCOUNT="$(gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null || true)"
[[ -n "$ACCOUNT" ]] || die "no active gcloud account; run 'gcloud auth login'"
log "project=$PROJECT account=$ACCOUNT"

if ! gcloud billing projects describe "$PROJECT" --format='value(billingEnabled)' 2>/dev/null | grep -qi true; then
  warn "billing not enabled on $PROJECT — Gemini paid tier calls will fail until billing is linked"
fi

# --- enable required services (idempotent) ---------------------------------
log "ensuring apikeys.googleapis.com + $GL_SERVICE are enabled"
gcloud services enable apikeys.googleapis.com "$GL_SERVICE" --project "$PROJECT" >/dev/null

# --- create or reuse the API key -------------------------------------------
KEY_NAME="$(gcloud services api-keys list --project "$PROJECT" \
  --filter="displayName=$KEY_DISPLAY_NAME" --format='value(name)' 2>/dev/null | head -1)"

if [[ -n "$KEY_NAME" ]]; then
  log "reusing existing key '$KEY_DISPLAY_NAME' ($KEY_NAME)"
else
  log "creating restricted API key '$KEY_DISPLAY_NAME'"
  KEY_NAME="$(gcloud services api-keys create \
    --display-name="$KEY_DISPLAY_NAME" \
    --api-target=service="$GL_SERVICE" \
    --project "$PROJECT" \
    --format='value(name)')"
fi
[[ -n "$KEY_NAME" ]] || die "failed to obtain key resource name"

KEY_STRING="$(gcloud services api-keys get-key-string "$KEY_NAME" --format='value(keyString)')"
[[ "$KEY_STRING" == AIza* ]] || die "unexpected key string format"
log "key string: ${KEY_STRING:0:6}…${KEY_STRING: -4}"

# --- smoke test ------------------------------------------------------------
# New keys can take a couple of minutes to propagate to the generateContent
# backends (transient HTTP 400 "API Key not found"); retry a few times.
log "testing $TEST_MODEL:generateContent (allowing for key propagation)"
test_status=""
for attempt in 1 2 3 4 5 6; do
  resp="$(curl -s -w $'\n%{http_code}' \
    "https://$GL_SERVICE/v1beta/models/$TEST_MODEL:generateContent?key=$KEY_STRING" \
    -H 'Content-Type: application/json' \
    -d '{"contents":[{"parts":[{"text":"Reply with exactly: OK"}]}]}')"
  code="${resp##*$'\n'}"
  body="${resp%$'\n'*}"
  case "$code" in
    200) log "✅ generateContent OK (HTTP 200) — key is fully working"; test_status=ok; break ;;
    429) warn "HTTP 429 — key authenticates, but Gemini prepaid credits are depleted."
         warn "Top up at https://ai.studio/projects (billing) to enable generateContent."
         test_status=credits; break ;;
    400) if grep -q 'API_KEY_INVALID\|API Key not found' <<<"$body"; then
           warn "attempt $attempt/6: key still propagating, retrying in 20s…"; sleep 20; continue
         fi
         warn "HTTP 400: $body"; test_status=err; break ;;
    403) die "HTTP 403 — project denied/suspended or key misrestricted: $body" ;;
    *)   warn "HTTP $code: $body"; test_status=err; break ;;
  esac
done
[[ -n "$test_status" ]] || warn "key did not finish propagating within retry window; try the test again shortly"

# --- optionally persist to .env --------------------------------------------
if [[ "$WRITE_ENV" == 1 ]]; then
  [[ -f "$ENV_FILE" ]] || die ".env not found at $ENV_FILE"
  backup="$ENV_FILE.bak-provision-$(date +%Y%m%d%H%M%S)"
  cp "$ENV_FILE" "$backup"
  log "backed up .env -> $(basename "$backup")"
  if grep -q '^GEMINI_API_KEY=' "$ENV_FILE"; then
    # portable in-place replace
    tmp="$(mktemp)"
    sed "s|^GEMINI_API_KEY=.*|GEMINI_API_KEY=\"$KEY_STRING\"|" "$ENV_FILE" >"$tmp" && mv "$tmp" "$ENV_FILE"
  else
    printf '\nGEMINI_API_KEY="%s"\n' "$KEY_STRING" >>"$ENV_FILE"
  fi
  log "wrote GEMINI_API_KEY to .env"
fi

echo
log "done. key resource: $KEY_NAME"
[[ "$WRITE_ENV" == 1 ]] || echo "  export GEMINI_API_KEY=\"$KEY_STRING\"   # or re-run with --write to update .env"
