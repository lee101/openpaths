#!/usr/bin/env python3
"""Rotate OPENROUTER_API_KEY across every consumer on this host.

Reuses the OpenRouter primitives in rotate_provider_key.py (create/validate/
list/delete) and adds the fan-out this host needs:

  1. mint + validate a replacement key
  2. write it into ~/.secretbashrc (the shell source every service inherits)
  3. write it into each repo .env that carries the key
  4. optionally delete every *other* key in the OpenRouter account
  5. print the services that must be restarted to pick it up

Nothing is written unless the new key validates against the OpenRouter API.

    export OPENROUTER_PROVISIONING_KEY=...        # OpenRouter dashboard -> Provisioning
    python3 rotation/rotate_openrouter_everywhere.py --dry-run
    python3 rotation/rotate_openrouter_everywhere.py
    python3 rotation/rotate_openrouter_everywhere.py --purge-others
"""

from __future__ import annotations

import argparse
import re
import shutil
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from rotate_provider_key import (  # noqa: E402
    RotationError,
    create_openrouter_key,
    list_openrouter_keys,
    openrouter_provisioning_key,
    parse_env_file,
    request_json,
)

ENV_KEY = "OPENROUTER_API_KEY"
SECRET_RC = Path.home() / ".secretbashrc"
CODE_ROOTS = [Path("/nvme0n1-disk/code"), Path("/sdb-disk/code")]

# Repo .env files that feed a running service. Backups (.bak*, .last) and
# .env.example files are deliberately skipped: rewriting a backup defeats its
# purpose, and examples must keep their placeholder.
ENV_FILE_GLOBS = ["*/.env", "*/.env.local", "*/.env.*.local", "*/*/.env", "*/*/.env.*.local"]
SKIP_PATTERNS = (".example", ".bak", ".last", ".migrated", "node_modules")


def log(msg: str) -> None:
    print(msg, flush=True)


def redact(secret: str) -> str:
    return secret[:12] + "..." + secret[-4:] if len(secret) > 20 else "***"


def discover_env_files() -> list[Path]:
    found: list[Path] = []
    for root in CODE_ROOTS:
        if not root.is_dir():
            continue
        for pattern in ENV_FILE_GLOBS:
            for path in root.glob(pattern):
                text = str(path)
                if any(s in text for s in SKIP_PATTERNS):
                    continue
                try:
                    if ENV_KEY in path.read_text(errors="ignore"):
                        found.append(path)
                except OSError:
                    continue
    return sorted(set(found))


def scan_hardcoded() -> list[Path]:
    """Source files with a literal sk-or-v1- key. These do not rotate."""
    found: list[Path] = []
    for root in CODE_ROOTS:
        if not root.is_dir():
            continue
        for path in list(root.glob("*/*.py")) + list(root.glob("*/*.go")) + list(root.glob("*/*.ts")):
            if any(s in str(path) for s in SKIP_PATTERNS):
                continue
            try:
                if "sk-or-v1-" in path.read_text(errors="ignore"):
                    found.append(path)
            except OSError:
                continue
    return sorted(set(found))


def replace_in_file(path: Path, new_secret: str, dry_run: bool) -> bool:
    """Rewrite every OPENROUTER_API_KEY assignment in path. Returns True if changed."""
    try:
        original = path.read_text()
    except OSError as exc:
        log(f"  ! cannot read {path}: {exc}")
        return False

    # Matches `KEY=value`, `export KEY=value`, quoted or bare.
    pattern = re.compile(
        rf"^(\s*(?:export\s+)?{re.escape(ENV_KEY)}\s*=\s*)(['\"]?)([^'\"\n]*)(\2)\s*$",
        re.MULTILINE,
    )
    matches = pattern.findall(original)
    if not matches:
        return False

    def sub(m: re.Match[str]) -> str:
        quote = m.group(2) or '"'
        return f"{m.group(1)}{quote}{new_secret}{quote}"

    updated = pattern.sub(sub, original)
    if updated == original:
        return False

    if dry_run:
        log(f"  would update {path} ({len(matches)} assignment(s))")
        return True

    backup = path.with_suffix(path.suffix + f".bak-rotation-{time.strftime('%Y%m%d%H%M%S')}")
    shutil.copy2(path, backup)
    path.write_text(updated)
    log(f"  updated {path} ({len(matches)} assignment(s), backup {backup.name})")
    return True


def purge_other_keys(provisioning_key: str, keep_hash: str, dry_run: bool) -> None:
    keys = list_openrouter_keys(provisioning_key)
    others = [k for k in keys if str(k.get("hash") or "") != keep_hash]
    if not others:
        log("  no other keys in the account")
        return
    for key in others:
        key_hash = str(key.get("hash") or "")
        label = str(key.get("label") or key.get("name") or "(unlabelled)")
        if not key_hash:
            continue
        if dry_run:
            log(f"  would DELETE {label}")
            continue
        try:
            request_json(
                "DELETE",
                f"https://openrouter.ai/api/v1/keys/{key_hash}",
                {"Authorization": f"Bearer {provisioning_key}"},
            )
            log(f"  deleted {label}")
        except Exception as exc:  # noqa: BLE001 - best effort, keep going
            log(f"  ! failed to delete {label}: {exc}")


def restart_hint() -> None:
    log("")
    log("Services inherit the key at start-up. Restart these to pick it up:")
    log("  netwrck        search_server_go/bin/netwrckprod27   (:8123)")
    log("  app.nz         app-site/server/appnz-server         (:8787)")
    log("  app.nz beta    app-site/server/appnz-beta           (:8788)")
    log("  chatgibidy     hiresnz2/chatgibidy/chatgibidy       (:8765)")
    log("  openpaths      openpaths-api                        (:8092)")
    log("  text-generator omniserve uvicorn                    (:8791)")
    log("  twohelixes     twohelixes-server                    (:7474)")
    log("")
    log("Off-host consumers are NOT covered by this script:")
    log("  - openpaths prod (93.127.141.100) - ./deploy.sh env")
    log("  - Evangeler prod (Starlink/Auckland box)")
    log("  - codex-infinity frozen standby")
    log("  - any k8s secret or CI secret holding the key")
    log("  Update those BEFORE --purge-others, or they break.")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true", help="show what would change, write nothing")
    parser.add_argument(
        "--purge-others",
        action="store_true",
        help="delete every other key in the OpenRouter account after installing the new one",
    )
    parser.add_argument("--alias", default=None, help="label for the new key")
    args = parser.parse_args()

    env_file_values = parse_env_file(Path(__file__).resolve().parents[1] / ".env")

    try:
        provisioning_key = openrouter_provisioning_key(env_file_values)
    except RotationError as exc:
        log(f"error: {exc}")
        log("Create one at https://openrouter.ai/settings/provisioning-keys and export it as")
        log("OPENROUTER_PROVISIONING_KEY. It can only manage keys, not spend.")
        return 2

    alias = args.alias or f"rotated-{time.strftime('%Y%m%d-%H%M%S')}"
    log(f"Minting replacement key ({alias})...")
    result = create_openrouter_key(alias, env_file_values)
    secret = result.secret
    new_hash = str(result.metadata.get("hash") or "")
    log(f"  minted + validated {redact(secret)}")

    hardcoded = scan_hardcoded()
    if hardcoded:
        log("\nWARNING - key hardcoded in source (this script cannot rotate it):")
        for path in hardcoded:
            log(f"  {path}")
        log("  Change these to read os.environ before purging old keys.")

    targets = [SECRET_RC] + discover_env_files()
    log(f"\nPropagating to {len(targets)} file(s):")
    changed = 0
    for path in targets:
        if replace_in_file(path, secret, args.dry_run):
            changed += 1
    log(f"  {changed} file(s) {'would be ' if args.dry_run else ''}updated")

    if args.purge_others:
        log("\nPurging other keys in the account:")
        purge_other_keys(provisioning_key, new_hash, args.dry_run)
    else:
        log("\nOther keys left in place (pass --purge-others to delete them).")

    if args.dry_run:
        log("\nDRY RUN - nothing was written. The minted key above is real and now")
        log("exists in the account; delete it in the dashboard if you are only testing.")

    restart_hint()
    return 0


if __name__ == "__main__":
    sys.exit(main())
