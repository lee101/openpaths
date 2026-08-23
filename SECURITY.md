# Secret scanning

Install Gitleaks and run `./scripts/install-gitleaks-hook.sh`. CI and the
pre-push hook scan with `.gitleaks.toml`. New OpenPaths keys use `sk-op-`; legacy `op-` keys remain accepted while they are rotated. Never commit `.env` files
or live credentials; revoke any credential found in Git.

