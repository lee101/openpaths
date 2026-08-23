#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
if ! command -v gitleaks >/dev/null 2>&1; then
  echo 'gitleaks is required for the pre-push secret scan.' >&2
  echo 'Install it from https://github.com/gitleaks/gitleaks#installation, then rerun this script.' >&2
  exit 1
fi

git config core.hooksPath .githooks
echo "Installed gitleaks pre-push hook for $root."
echo 'The hook scans commits being pushed; CI also scans every push and pull request.'
