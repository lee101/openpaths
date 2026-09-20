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
