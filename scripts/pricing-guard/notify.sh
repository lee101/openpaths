#!/usr/bin/env bash
# notify.sh SUBJECT [BODY|-]  - send an alert email through Amazon SES SMTP.
# BODY "-" (or omitted) reads the body from stdin.
# From lee.penkman@netwrck.com to leepenkman@gmail.com by default; SMTP creds are
# read from the environment or the first env file that has them (never printed).
set -euo pipefail

SUBJECT=${1:-"openpaths pricing guard"}
if [ "${2:--}" = "-" ]; then BODY=$(cat); else BODY=$2; fi
TO=${ALERT_EMAIL:-leepenkman@gmail.com}
FROM=${ALERT_FROM:-lee.penkman@netwrck.com}
ENV_FILES=${ALERT_ENV_FILES:-"$HOME/.secretbashrc /nvme0n1-disk/code/openpaths/.env /nvme0n1-disk/code/netwrck/.env"}

env_value() { # env_value NAME -> first value found across ENV_FILES
    local f v
    for f in $ENV_FILES; do
        [ -r "$f" ] || continue
        v=$(sed -n -E "s/^(export )?$1=['\"]?([^'\"]*)['\"]?$/\2/p" "$f" | tail -1)
        [ -n "$v" ] && { printf '%s' "$v"; return; }
    done
}

REGION=${AWS_REGION:-$(env_value AWS_REGION)}; REGION=${REGION:-us-east-1}
SMTP_USER=${AWS_SMTP_USERNAME:-$(env_value AWS_SMTP_USERNAME)}
SMTP_PASS=${AWS_SMTP_PASSWORD:-$(env_value AWS_SMTP_PASSWORD)}
if [ -z "$SMTP_USER" ] || [ -z "$SMTP_PASS" ]; then
    echo "notify: no SES SMTP credentials found in env or: $ENV_FILES" >&2
    exit 2
fi

HOST="email-smtp.$REGION.amazonaws.com"
NETRC=$(mktemp); MESSAGE=$(mktemp)
trap 'rm -f "$NETRC" "$MESSAGE"' EXIT
chmod 600 "$NETRC" "$MESSAGE"
printf 'machine %s login %s password %s\n' "$HOST" "$SMTP_USER" "$SMTP_PASS" > "$NETRC"
{
    printf 'From: %s\r\nTo: %s\r\nSubject: %s\r\n' "$FROM" "$TO" "$SUBJECT"
    printf 'Date: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n' "$(date -R)"
    printf '%s\r\n' "$BODY"
} > "$MESSAGE"
curl -fsS --max-time 60 --ssl-reqd --url "smtp://$HOST:587" --netrc-file "$NETRC" \
    --mail-from "$FROM" --mail-rcpt "$TO" --upload-file "$MESSAGE"
