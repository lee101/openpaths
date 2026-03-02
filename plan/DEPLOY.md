# OpenPath Deployment

## Server

- Host: `administrator@93.127.141.100`
- Path: `/nvme0n1-disk/code/openpath`
- API port: `8080` (behind nginx)
- Process manager: supervisor (`openpath`)
- Static CDN: `openpathsstatic.openpaths.io` (Cloudflare R2)

## SSH

```bash
# Key-based auth (preferred):
alias sscp='ssh administrator@93.127.141.100'

# Or with sshpass if needed:
# export SSH_PASS=<password>
# alias sscp='sshpass -p "$SSH_PASS" ssh -o StrictHostKeyChecking=no administrator@93.127.141.100'
```

## deploy.sh

```bash
./deploy.sh site     # build frontend + sync to Cloudflare R2
./deploy.sh api      # build Go binary + deploy to server
./deploy.sh env      # sync .env to server
./deploy.sh setup    # create dirs + supervisor config (first time)
./deploy.sh all      # site + api
```

Requires env vars: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`

Optional: `CLOUDFLARE_EMAIL`, `CLOUDFLARE_API_KEY`, `CLOUDFLARE_ZONE_OPENPATHS` for cache purge.

## First-time Setup

```bash
# 1. Setup server directories + supervisor
./deploy.sh setup

# 2. Deploy .env with API keys
./deploy.sh env

# 3. Deploy everything
./deploy.sh all
```

## Quick Deploy (manual)

```bash
# 1. Build frontend
npm run build

# 2. Build Go API for linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/openpath-api ./cmd/openpath/

# 3. SCP binary to remote
sshpass -p "$SSH_PASS" scp -o StrictHostKeyChecking=no \
  dist/openpath-api administrator@93.127.141.100:/nvme0n1-disk/code/openpath/openpath-api.new

# 4. Rsync dist/ to remote
sshpass -p "$SSH_PASS" rsync -az --delete \
  -e 'sshpass -p ka3iMI4OSNvgFcREuDaQyLguFxuP ssh -o StrictHostKeyChecking=no' \
  --exclude='openpath-api*' dist/ administrator@93.127.141.100:/nvme0n1-disk/code/openpath/dist/

# 5. Swap binary + restart
sscp 'cd /nvme0n1-disk/code/openpath && mv openpath-api.new openpath-api && chmod +x openpath-api && sudo supervisorctl restart openpath'
```

## Git-based Deploy (on server)

```bash
sscp
cd /nvme0n1-disk/code/openpath
git pull origin master
npm install && npm run build
go build -o openpath-api ./cmd/openpath/
sudo supervisorctl restart openpath
```

## Supervisor

Config: `/etc/supervisor/conf.d/openpath.conf`

```ini
[program:openpath]
command=/nvme0n1-disk/code/openpath/openpath-api
directory=/nvme0n1-disk/code/openpath
user=administrator
autostart=true
autorestart=true
stderr_logfile=/var/log/supervisor/openpath-error.log
stdout_logfile=/var/log/supervisor/openpath.log
stdout_logfile_maxbytes=10MB
stdout_logfile_backups=3
```

Commands:
```bash
sudo supervisorctl status openpath
sudo supervisorctl restart openpath
sudo supervisorctl stop openpath
sudo supervisorctl tail -f openpath
```

## Production .env (on server)

```
PORT=8080
DATABASE_URL=postgres://openpath:openpath@localhost:5432/openpath?sslmode=disable
JWT_SECRET=<random-secret>
OPENAI_API_KEY=...
ANTHROPIC_API_KEY=...
GEMINI_API_KEY=...
GROQ_API_KEY=...
XAI_API_KEY=...
DEEPSEEK_API_KEY=...
OPENROUTER_API_KEY=...
TOGETHER_API_KEY=...
MINIMAX_API_KEY=...
CRYPTO_ENABLED=true
CRYPTO_SOLANA_RPC_URL=...
CRYPTO_WALLET_PUBKEY=...
```

## Architecture

- Go API serves both API endpoints and static frontend from `dist/`
- nginx reverse proxies to `:8080`
- Static assets also served from R2 CDN at `openpathsstatic.openpaths.io`
- Frontend is a Vite/React SPA
- Database: PostgreSQL on localhost

## E2E Tests

```bash
npm run build
npm run test:e2e        # run all tests
npm run test:e2e:ui     # interactive UI mode
```

55 tests across 4 suites: navigation, landing, models, playground, account (overview/keys/billing), Stripe portal, Stripe modal, usage graph.
