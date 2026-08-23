#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROTATION_DIR="$ROOT/rotation"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if [[ "${1:-}" == "--init" ]]; then
    echo "This imports management keys from .env, encrypts them, and removes them from .env."
    read -r -s -p "Encryption password (use your chosen passphrase, e.g. Leap): " password
    echo
    read -r -s -p "Confirm encryption password: " confirm
    echo
    [[ "$password" == "$confirm" ]] || { echo "Passwords do not match" >&2; exit 1; }
    [[ -n "$password" ]] || { echo "Password must not be empty" >&2; exit 1; }

    python3 "$ROTATION_DIR/secret_bundle.py" extract "$ROOT/.env" "$TMP_DIR"
    for provider in openai openrouter; do
        printf '%s\n' "$password" | openssl enc -aes-256-cbc -salt -pbkdf2 -iter 600000 \
            -pass stdin \
            -in "$TMP_DIR/${provider}.env" -out "$ROTATION_DIR/.envsecret.$provider"
    done
    python3 "$ROTATION_DIR/secret_bundle.py" remove "$ROOT/.env"
    chmod 600 "$ROTATION_DIR/.envsecret."*
    echo "Encrypted management bundles created; management keys removed from .env."
    exit 0
fi

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    exec python3 "$ROTATION_DIR/rotate_provider_key.py" all --fanout --help
fi

read -r -s -p "Decryption password: " password
echo
[[ -n "$password" ]] || { echo "Password must not be empty" >&2; exit 1; }

for provider in openai openrouter; do
    printf '%s\n' "$password" | openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 \
        -pass stdin \
        -in "$ROTATION_DIR/.envsecret.$provider" -out "$TMP_DIR/$provider.env"
done
cat "$TMP_DIR/openai.env" "$TMP_DIR/openrouter.env" > "$TMP_DIR/management.env"

set -a
# shellcheck disable=SC1091
source "$TMP_DIR/management.env"
set +a
exec env -u ROTATION_PASSWORD python3 "$ROTATION_DIR/rotate_provider_key.py" all --fanout "$@"
