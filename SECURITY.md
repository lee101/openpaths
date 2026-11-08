# Secret scanning

Install Gitleaks and run `./scripts/install-gitleaks-hook.sh`. CI and the
pre-push hook scan with `.gitleaks.toml`. New OpenPaths keys use `sk-op-`; legacy `op-` keys remain accepted while they are rotated. Never commit `.env` files
or live credentials; revoke any credential found in Git.

The pre-push hook scans outgoing commits once per push, excluding history
already known on the destination remote for new branches. Branch deletions
need no scan. Ignored local files such as `.env`, dependencies, and build output
are not part of the push scan. If remote history is unavailable, the hook scans
the outgoing history conservatively. Run `make install-hooks` to activate this
repository's hook, replacing any previously configured hook path for this repo.

# API key scopes

Keys carry scopes (`GET /v1/keys/scopes`). Existing keys with no scopes keep full usage access but cannot manage keys; `keys:manage` must be granted explicitly. Every key-authenticated route declares a scope (enforced by a test in `internal/server`), and a route without one fails closed with 403. Keys can expire, be revoked, and be rotated with a grace period (`POST /v1/keys/{id}/rotate`). Secrets are shown once and stored only as hashes. `rotation/rotate_openpaths_keys.py` bulk-rotates an account's keys.
