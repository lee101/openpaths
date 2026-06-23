#!/usr/bin/env bash
# Fix Application Default Credentials (ADC).
#
# Symptom this fixes:
#   ERROR: (gcloud.auth.application-default.print-access-token)
#   File /home/lee/Downloads/questions-*.json was not found.
#
# Cause: GOOGLE_APPLICATION_CREDENTIALS is exported to a stale service-account
# path that no longer exists, which overrides real ADC.
#
# OpenPaths talks to Gemini via an API key (GEMINI_API_KEY), so ADC is NOT
# required for the app. Only run this if you need ADC for other Google libraries
# (Vertex AI, GCS, etc.).
set -euo pipefail

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }

if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" && ! -f "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  log "GOOGLE_APPLICATION_CREDENTIALS points at a missing file:"
  log "  ${GOOGLE_APPLICATION_CREDENTIALS}"
  log "Unset it in your shell/profile, e.g.:"
  echo "    unset GOOGLE_APPLICATION_CREDENTIALS"
  echo "  and remove any 'export GOOGLE_APPLICATION_CREDENTIALS=...' from ~/.bashrc / .env"
fi

log "To (re)establish user ADC, run interactively:"
echo "    gcloud auth application-default login"
echo "    gcloud auth application-default set-quota-project \"\$(gcloud config get-value project)\""
log "Then verify:"
echo "    gcloud auth application-default print-access-token | head -c 20; echo"
