#!/usr/bin/env python3
"""Extract and remove the small set of rotation-only management variables."""

from pathlib import Path
import re
import sys

KEYS = {
    "OPENAI_ADMIN_KEY": "openai",
    "OPENROUTER_MANAGEMENT_KEY": "openrouter",
}


def values(path: Path) -> dict[str, str]:
    result: dict[str, str] = {}
    for line in path.read_text().splitlines():
        match = re.match(r"^\s*(?:export\s+)?([A-Z][A-Z0-9_]*)\s*=\s*(.*)\s*$", line)
        if not match or match.group(1) not in KEYS:
            continue
        raw = match.group(2)
        if len(raw) >= 2 and raw[0] in "'\"" and raw[-1] == raw[0]:
            raw = raw[1:-1]
        result[match.group(1)] = raw
    return result


def extract(source: Path, output_dir: Path) -> None:
    found = values(source)
    missing = [key for key in KEYS if not found.get(key)]
    if missing:
        raise SystemExit("missing management key(s): " + ", ".join(missing))
    output_dir.mkdir(parents=True, exist_ok=True)
    for key, provider in KEYS.items():
        (output_dir / f"{provider}.env").write_text(f"{key}={found[key]!r}\n")


def remove(path: Path) -> None:
    lines = path.read_text().splitlines()
    kept = []
    for line in lines:
        match = re.match(r"^\s*(?:export\s+)?([A-Z][A-Z0-9_]*)\s*=", line)
        if match and match.group(1) in KEYS:
            continue
        kept.append(line)
    path.write_text("\n".join(kept) + "\n")


if __name__ == "__main__":
    if len(sys.argv) < 3 or sys.argv[1] not in {"extract", "remove"} or (sys.argv[1] == "extract" and len(sys.argv) != 4) or (sys.argv[1] == "remove" and len(sys.argv) != 3):
        raise SystemExit("usage: secret_bundle.py {extract|remove} .env [output-dir for extract]")
    if sys.argv[1] == "extract":
        extract(Path(sys.argv[2]), Path(sys.argv[3]))
    else:
        remove(Path(sys.argv[2]))
