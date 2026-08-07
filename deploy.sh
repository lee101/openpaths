#!/usr/bin/env bash
set -euo pipefail

# Load .env so cache-purge creds (CLOUDFLARE_EMAIL/_API_KEY/_ZONE_*) and R2 keys
# are present even in a non-interactive shell (cron, CI). Shell env still wins.
if [[ -f .env ]]; then
    set -a
    # shellcheck disable=SC1091
    source .env
    set +a
fi

# --- Config ---
STATIC_BUCKET="${STATIC_BUCKET:-openpathsstatic}"
R2_ENDPOINT="${R2_ENDPOINT:-https://${R2_ACCOUNT_ID:-f76d25b8b86cfa5638f43016510d8f77}.r2.cloudflarestorage.com}"
STATIC_URL="${STATIC_URL:-https://openpathsstatic.openpaths.io}"
API_HOST="${API_HOST:-openpaths-prod}"
API_SERVICE="openpaths"
REMOTE_DIR="/nvme0n1-disk/code/openpaths"
DIST_DIR="dist"
CF_ZONE_ID="${CLOUDFLARE_ZONE_OPENPATHS:-${CLOUDFLARE_ZONE_OPENPATH:-}}"
SSH_PASS="${SSH_PASS:-}"
API_HEALTH_URL="${API_HEALTH_URL:-http://127.0.0.1:8092/health}"
API_DRAIN_TIMEOUT_SECONDS="${API_DRAIN_TIMEOUT_SECONDS:-600}"

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

is_local_api_host() {
    local host="${API_HOST#*@}"
    if command -v ssh >/dev/null 2>&1; then
        local ssh_host
        ssh_host=$(ssh -G "${API_HOST}" 2>/dev/null | awk '$1 == "hostname" { print $2; exit }' || true)
        if [[ -n "$ssh_host" ]]; then
            host="$ssh_host"
        fi
    fi
    case "$host" in
        localhost|127.0.0.1|::1|"$(hostname)"|"$(hostname -f 2>/dev/null || true)")
            return 0
            ;;
    esac

    local local_ips
    local_ips=" $(hostname -I 2>/dev/null || true) "
    if [[ -n "$host" && "$local_ips" == *" ${host} "* ]]; then
        return 0
    fi

    if command -v getent >/dev/null 2>&1; then
        local ip
        while read -r ip _; do
            [[ -n "$ip" ]] || continue
            if ip route get "$ip" 2>/dev/null | grep -q " dev lo "; then
                return 0
            fi
        done < <(getent ahostsv4 "$host" 2>/dev/null || true)
    fi

    return 1
}

restart_api_local() {
    wait_for_local_api_idle
    cd "${REMOTE_DIR}"
    mv openpaths-api.new openpaths-api
    chmod +x openpaths-api
    if sudo supervisorctl status "${API_SERVICE}" >/dev/null 2>&1; then
        sudo supervisorctl restart "${API_SERVICE}"
    else
        kill "$(pgrep -x openpaths-api)" 2>/dev/null || true
        sleep 1
        nohup ./openpaths-api > "/var/log/supervisor/${API_SERVICE}.log" 2>&1 &
    fi
    echo "api deployed"
}

wait_for_local_api_idle() {
    local deadline=$((SECONDS + API_DRAIN_TIMEOUT_SECONDS))
    while true; do
        local response active
        response=$(curl -fsS "${API_HEALTH_URL}" 2>/dev/null || true)
        active=$(python3 -c 'import json,sys; d=json.loads(sys.stdin.read() or "{}"); print(int(d.get("active_long_requests", 0)))' <<<"$response" 2>/dev/null || echo 0)
        if [[ "$active" -eq 0 ]]; then
            return
        fi
        if (( SECONDS >= deadline )); then
            yellow "continuing deploy with ${active} long request(s) still active after ${API_DRAIN_TIMEOUT_SECONDS}s"
            return
        fi
        yellow "waiting for ${active} active video/3d request(s) before restart..."
        sleep 5
    done
}

FRONTEND_BUILT=false

build_frontend() {
    if [[ "$FRONTEND_BUILT" == true ]]; then
        return
    fi

    green "cleaning frontend output..."
    npm run clean

    green "building frontend..."
    npm run build

    green "verifying stable asset names..."
    npm run verify:assets

    FRONTEND_BUILT=true
}

purge_cf_cache() {
    local email="${CLOUDFLARE_EMAIL:-}"
    local key="${CLOUDFLARE_API_KEY:-}"
    local zone="${CF_ZONE_ID:-}"
    if [[ -z "$email" || -z "$key" ]]; then
        yellow "cloudflare cache purge skipped (missing CLOUDFLARE_EMAIL / CLOUDFLARE_API_KEY)"
        return
    fi
    # Auto-resolve the zone id from the API when CLOUDFLARE_ZONE_OPENPATHS is unset,
    # so a missing env var never silently skips the purge (the cause of stale
    # sitemaps/bundles lingering after a deploy).
    if [[ -z "$zone" ]]; then
        green "resolving cloudflare zone id for openpaths.io..."
        zone=$(curl -fsS "https://api.cloudflare.com/client/v4/zones?name=openpaths.io" \
            -H "X-Auth-Email: ${email}" -H "X-Auth-Key: ${key}" \
            | grep -oE '"id":"[a-f0-9]{32}"' | head -1 | cut -d'"' -f4)
    fi
    if [[ -z "$zone" ]]; then
        red "cloudflare cache purge FAILED: could not resolve zone id for openpaths.io"
        exit 1
    fi
    green "purging cloudflare cache (zone ${zone})..."
    local response
    response=$(curl -fsS -X POST \
        "https://api.cloudflare.com/client/v4/zones/${zone}/purge_cache" \
        -H "X-Auth-Email: ${email}" \
        -H "X-Auth-Key: ${key}" \
        -H "Content-Type: application/json" \
        --data '{"purge_everything":true}')
    if ! grep -q '"success":true' <<<"$response"; then
        red "cloudflare cache purge failed: $response"
        exit 1
    fi
}

upload_prerender_routes() {
    local manifest="${DIST_DIR}/prerender-manifest.json"
    if [[ ! -f "$manifest" ]]; then
        return
    fi

    green "uploading prerendered route metadata..."
    node -e '
const fs = require("fs");
const manifest = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
for (const route of manifest) {
  process.stdout.write(`${route.source}\t${route.key}\n`);
}
' "$manifest" | while IFS=$'\t' read -r source key; do
        [[ -n "$source" && -n "$key" ]] || continue
        aws s3 cp "$source" "s3://${STATIC_BUCKET}/${key}" \
            --endpoint-url "${R2_ENDPOINT}" \
            --content-type "text/html; charset=utf-8" \
            --cache-control "public, max-age=300" \
            >/dev/null
    done
}

# --- Site deploy: build frontend + sync to R2 ---
deploy_site() {
    build_frontend

    green "syncing to R2 (${STATIC_BUCKET})..."
    require_cmd aws

    aws s3 sync "${DIST_DIR}/" "s3://${STATIC_BUCKET}/" \
        --endpoint-url "${R2_ENDPOINT}" \
        --size-only \
        --delete \
        --exclude "static/data/zimage-art/*" \
        --exclude "static/uploads/*" \
        --exclude "uploads/*" \
        --exclude "*.map"

    upload_prerender_routes
    purge_cf_cache
    green "site deployed to ${STATIC_URL}"
}

# --- API deploy: build Go binary + deploy to server ---
deploy_api() {
    build_frontend

    green "building api..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/openpaths-api ./cmd/openpaths/

    green "deploying api to ${API_HOST}..."

    if is_local_api_host && [[ "$(pwd -P)" == "${REMOTE_DIR}" ]]; then
        green "api host is local; installing without ssh/scp..."
        cp dist/openpaths-api "${REMOTE_DIR}/openpaths-api.new"
        green "config already local: ${REMOTE_DIR}/config.yaml"
        green "swapping binary + restarting locally..."
        restart_api_local
        green "api deployed locally"
        purge_cf_cache
        return
    fi

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
    # The API serves dynamic content (sitemaps, /v1/art, og/meta) that Cloudflare
    # caches, so purge after the new binary is live or crawlers keep seeing stale XML.
    purge_cf_cache
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
    cache)   purge_cf_cache; green "cloudflare cache purged" ;;
    all)
        deploy_site
        deploy_api
        ;;
    *)
        echo "usage: $0 {site|api|env|setup|cache|all}"
        exit 1
        ;;
esac
