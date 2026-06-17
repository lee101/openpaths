#!/usr/bin/env bash
# fetch-gobed-model.sh — download the gobed int8 embedding model from the R2
# `models/dev/gobed/` prefix into a destination dir (default: server/model).
# The model is gitignored (15MB) and synced out-of-band via R2 instead of git.
#
#   scripts/fetch-gobed-model.sh                 # -> server/model/
#   scripts/fetch-gobed-model.sh model           # -> model/ (openpaths layout)
#   DEST=/path scripts/fetch-gobed-model.sh
#
# Reads R2 creds from the environment, falling back to ./.env (CLOUDFLARE_R2_*,
# R2_ACCOUNT_ID). Requires the AWS CLI.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
DEST="${1:-${DEST:-server/model}}"

val() { awk -F'"' "/^$1=/{print \$2; exit}" .env 2>/dev/null; }

AKID="${CLOUDFLARE_R2_ACCESS_KEY_ID:-$(val CLOUDFLARE_R2_ACCESS_KEY_ID)}"
SAK="${CLOUDFLARE_R2_SECRET_ACCESS_KEY:-$(val CLOUDFLARE_R2_SECRET_ACCESS_KEY)}"
ACCT="${R2_ACCOUNT_ID:-$(val R2_ACCOUNT_ID)}"
BUCKET="${MODELS_BUCKET:-models}"
PREFIX="${MODELS_PREFIX:-dev/gobed}"

if [[ -z "$AKID" || -z "$SAK" || -z "$ACCT" ]]; then
  echo "fetch-gobed-model: missing R2 creds (CLOUDFLARE_R2_ACCESS_KEY_ID/SECRET_ACCESS_KEY, R2_ACCOUNT_ID)" >&2
  exit 1
fi

EP="https://${ACCT}.r2.cloudflarestorage.com"
export AWS_ACCESS_KEY_ID="$AKID" AWS_SECRET_ACCESS_KEY="$SAK" AWS_DEFAULT_REGION=auto AWS_EC2_METADATA_DISABLED=true

mkdir -p "$DEST"
echo "==> fetching gobed model from s3://${BUCKET}/${PREFIX}/ -> ${DEST}/"
for f in modelint8_512dim.safetensors tokenizer.json; do
  aws s3 --endpoint-url "$EP" cp "s3://${BUCKET}/${PREFIX}/${f}" "${DEST}/${f}"
done
echo "==> done:"; ls -la "$DEST"
