# OpenPaths Deployment

This repository is open source. Do not commit production IPs, usernames, passwords, SSH commands with secrets, or private infrastructure paths.

## SSH

Prefer a local SSH alias:

```sshconfig
Host openpaths-prod
  HostName <your-prod-host>
  User <your-user>
  IdentityFile ~/.ssh/<your-key>
```

Or export a host at runtime:

```bash
export API_HOST=<user>@<host>
```

If you need password auth temporarily, set `SSH_PASS` locally. Never commit it.

## deploy.sh

```bash
./deploy.sh site
./deploy.sh api
./deploy.sh env
./deploy.sh setup
./deploy.sh all
```

Common environment variables:

```bash
export API_HOST=openpaths-prod
export REMOTE_DIR=/nvme0n1-disk/code/openpaths
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

Optional Cloudflare cache purge:

```bash
export CLOUDFLARE_EMAIL=...
export CLOUDFLARE_API_KEY=...
export CLOUDFLARE_ZONE_OPENPATHS=...
```

## First-Time Setup

```bash
./deploy.sh setup
./deploy.sh env
./deploy.sh all
```

## Runtime Expectations

- The app runs from the same directory as `.env`, `config.yaml`, and `dist/`.
- nginx serves static assets from `dist/` and proxies API routes to the app port.
- Use one canonical deployment directory and one supervisor unit name: `openpaths`.
- Keep `PORT`, `APP_URL`, Stripe secrets, and provider API keys in `.env`.

## Verification

After deploy:

```bash
ssh "$API_HOST" 'sudo supervisorctl status openpaths'
curl -fsS https://<your-domain>/health
curl -fsSI https://<your-domain>/
```

Frontend regression checks:

```bash
npm run build
npm run test:e2e
```
