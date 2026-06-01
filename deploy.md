# Deployment

## Prod Server

```
ssh -o StrictHostKeyChecking=no administrator@93.127.141.100
```

SSH config alias: `openpaths-prod`
Remote dir: `/nvme0n1-disk/code/openpaths`
Managed by: supervisord (service name: `openpaths`)

## deploy.sh

```bash
./deploy.sh site    # build frontend, sync to R2 (openpathsstatic bucket)
./deploy.sh api     # build Go binary, scp to prod, restart
./deploy.sh env     # sync .env to prod
./deploy.sh setup   # create remote dirs + supervisor config
./deploy.sh all     # site + api (default)
```

### What each target does

**site** - `npm run build`, then `aws s3 sync` dist to Cloudflare R2 bucket `openpathsstatic`, purges CF cache.

**api** - Cross-compiles `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/openpaths-api ./cmd/openpaths/`, scps binary + config.yaml + dist/ to prod, swaps binary and restarts via supervisorctl.

**env** - Copies local `.env` to prod server.

**setup** - Creates remote dir and supervisor config on first deploy.

## Required env vars for deploy

- `SSH_PASS` - if set, uses sshpass for auth (otherwise uses ssh keys)
- `R2_ACCOUNT_ID` / `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` - for R2 uploads
- `CLOUDFLARE_EMAIL` / `CLOUDFLARE_API_KEY` / `CLOUDFLARE_ZONE_OPENPATHS` - for cache purge

## Static assets

- Bucket: `openpathsstatic`
- URL: `https://openpathsstatic.openpaths.io`

## Frontend nginx / asset caching

- Prod nginx vhost is version-controlled at [deploy/nginx/openpaths.io](deploy/nginx/openpaths.io)
  (live copy: `/etc/nginx/sites-enabled/openpaths.io`). Edit, then `sudo nginx -t && sudo systemctl reload nginx`.
- Asset bundles use **stable** filenames (`/assets/index.js`, `index.css` -- enforced by
  `npm run verify:assets`). Because the filename never changes, the `/assets/` block serves
  `Cache-Control: public, max-age=0, must-revalidate` (NOT `immutable`) so a redeployed bundle is
  picked up on the next load via ETag revalidation. Never re-add `immutable` here -- it pins
  returning browsers to a stale bundle and breaks newly-added SPA routes ("No routes matched").
- The Cloudflare zone Browser Cache TTL is set to **Respect Existing Headers** so the origin
  `max-age=0` is honored. After deploying a new bundle, purge the asset URLs (or `deploy.sh site`
  purges everything) so the edge re-fetches.

## Prod DB

Postgres on the prod server. If you renamed the local DB from `openpath` to `openpaths`, the prod DB may still use the old name -- coordinate the rename on prod separately:
```sql
ALTER USER openpath RENAME TO openpaths;
ALTER USER openpaths WITH PASSWORD 'openpaths';
ALTER DATABASE openpath RENAME TO openpaths;
```
