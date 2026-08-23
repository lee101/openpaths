#!/usr/bin/env bash
set -euo pipefail
git config core.hooksPath .githooks
echo 'Installed .githooks as the repository hook path.'

