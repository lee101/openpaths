#!/usr/bin/env bash
set -euo pipefail

# --- Config ---
STATIC_BUCKET="${STATIC_BUCKET:-openpathsstatic}"
R2_ENDPOINT="${R2_ENDPOINT:-https://${R2_ACCOUNT_ID:-f76d25b8b86cfa5638f43016510d8f77}.r2.cloudflarestorage.com}"
STATIC_URL="${STATIC_URL:-https://openpathsstatic.openpaths.io}"
API_HOST="${API_HOST:-openpaths-prod}"
API_SERVICE="openpaths"
REMOTE_DIR="/nvme0n1-disk/code/openpaths"
DIST_DIR="dist"
CF_ZONE_ID="${CLOUDFLARE_ZONE_OPENPATHS:-}"
SSH_PASS="${SSH_PASS:-}"

export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-${CLOUDFLARE_R2_ACCESS_KEY_ID:-}}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-${CLOUDFLARE_R2_SECRET_ACCESS_KEY:-}}"

red()    { printf '\033[0;31m%s\033[0m\n' "$*"; }
green()  { printf '\033[0;32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[0;33m%s\033[0m\n' "$*"; }

require_cmd() {
    command -v "$1" >/dev/null 2>&1 || { red "missing: $1"; exit 1; }
}

require_sshpass_if_needed() {
    if [[ -n "$SSH_PASS" ]]; then
        require_cmd sshpass
    fi
}

ssh_cmd() {
    if [[ -n "$SSH_PASS" ]]; then
        require_sshpass_if_needed
        sshpass -p "${SSH_PASS}" ssh -o StrictHostKeyChecking=no "${API_HOST}" "$@"
    else
        ssh "${API_HOST}" "$@"
    fi
}

scp_cmd() {
    if [[ -n "$SSH_PASS" ]]; then
        require_sshpass_if_needed
        sshpass -p "${SSH_PASS}" scp -o StrictHostKeyChecking=no "$@"
    else
        scp "$@"
    fi
}

rsync_cmd() {
    if [[ -n "$SSH_PASS" ]]; then
        require_sshpass_if_needed
        sshpass -p "${SSH_PASS}" rsync -az \
            -e "sshpass -p ${SSH_PASS} ssh -o StrictHostKeyChecking=no" "$@"
    else
        rsync -az "$@"
    fi
}

purge_cf_cache() {
    local email="${CLOUDFLARE_EMAIL:-}"
    local key="${CLOUDFLARE_API_KEY:-}"
    if [[ -n "$email" && -n "$key" && -n "$CF_ZONE_ID" ]]; then
        green "purging cloudflare cache..."
        curl -s -X POST \
            "https://api.cloudflare.com/client/v4/zones/${CF_ZONE_ID}/purge_cache" \
            -H "X-Auth-Email: ${email}" \
            -H "X-Auth-Key: ${key}" \
            -H "Content-Type: application/json" \
            --data '{"purge_everything":true}' >/dev/null
    else
        yellow "cloudflare cache purge skipped (missing credentials or zone id)"
    fi
}

# --- Site deploy: build frontend + sync to R2 ---
deploy_site() {
    green "building frontend..."
    npm run build

    green "syncing to R2 (${STATIC_BUCKET})..."
    require_cmd aws

    aws s3 sync "${DIST_DIR}/" "s3://${STATIC_BUCKET}/" \
        --endpoint-url "${R2_ENDPOINT}" \
        --size-only \
        --delete \
        --exclude "static/uploads/*" \
        --exclude "uploads/*" \
        --exclude "*.map"

    purge_cf_cache
    green "site deployed to ${STATIC_URL}"
}

# --- API deploy: build Go binary + deploy to server ---
deploy_api() {
    green "building api..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/openpaths-api ./cmd/openpaths/

    green "deploying api to ${API_HOST}..."

    scp_cmd dist/openpaths-api "${API_HOST}:${REMOTE_DIR}/openpaths-api.new"

    green "syncing frontend dist..."
    rsync_cmd --delete --exclude='openpaths-api*' dist/ "${API_HOST}:${REMOTE_DIR}/dist/"

    green "syncing config..."
    scp_cmd config.yaml "${API_HOST}:${REMOTE_DIR}/config.yaml"

    green "swapping binary + restarting..."
    ssh_cmd bash -s <<REMOTE
        set -euo pipefail
        cd ${REMOTE_DIR}
        mv openpaths-api.new openpaths-api
        chmod +x openpaths-api
        if sudo supervisorctl status ${API_SERVICE} >/dev/null 2>&1; then
            sudo supervisorctl restart ${API_SERVICE}
        else
            kill \$(pgrep -x openpaths-api) 2>/dev/null || true
            sleep 1
            nohup ./openpaths-api > /var/log/supervisor/${API_SERVICE}.log 2>&1 &
        fi
        echo "api deployed"
REMOTE

    green "api deployed to ${API_HOST}"
}

# --- Deploy env file ---
deploy_env() {
    green "syncing .env to server..."
    scp_cmd .env "${API_HOST}:${REMOTE_DIR}/.env"
    green ".env deployed"
}

# --- Server setup: create dirs + supervisor config ---
deploy_setup() {
    green "setting up server..."

    ssh_cmd bash -s <<REMOTE
        set -euo pipefail
        mkdir -p ${REMOTE_DIR}/dist
        echo "directory created: ${REMOTE_DIR}"

        if ! sudo test -f /etc/supervisor/conf.d/${API_SERVICE}.conf; then
            sudo tee /etc/supervisor/conf.d/${API_SERVICE}.conf > /dev/null <<'CONF'
[program:${API_SERVICE}]
command=${REMOTE_DIR}/openpaths-api
directory=${REMOTE_DIR}
user=administrator
autostart=true
autorestart=true
stderr_logfile=/var/log/supervisor/${API_SERVICE}-error.log
stdout_logfile=/var/log/supervisor/${API_SERVICE}.log
stdout_logfile_maxbytes=10MB
stdout_logfile_backups=3
CONF
            sudo supervisorctl reread
            sudo supervisorctl update
            echo "supervisor config created"
        else
            echo "supervisor config already exists"
        fi
REMOTE
    green "server setup complete"
}

# --- Main ---
CMD="${1:-all}"

case "$CMD" in
    site)    deploy_site ;;
    api)     deploy_api ;;
    env)     deploy_env ;;
    setup)   deploy_setup ;;
    all)
        deploy_site
        deploy_api
        ;;
    *)
        echo "usage: $0 {site|api|env|setup|all}"
        exit 1
        ;;
esac
