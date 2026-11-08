#!/usr/bin/env python3
"""Rotate every OpenPaths API key owned by one account (default leepenkman@gmail.com).

Keys are stored hashed (sha256) in prod's api_keys table, so we find where each key
lives by hashing every `op-…` / `sk-op-…` token found in env/config/shell files on this
machine and on prod and matching against the account's active key hashes.

Phases (nothing changes without a flag):
  (default)   dry run: print what was found and what would happen
  --apply     mint a replacement per found key (same name + rate limit), swap it into every
              file that held the old one (local + prod, incl. ~/.secretbashrc), write
              rotation/.openpaths-rotation-state.json. Old keys stay valid so running
              services keep working until restarted.
  --finalize  after restarting dependent services: revoke the old keys AND every other
              active key of the account that was not found anywhere (rows are revoked, not
              deleted, because usage history references them).

Only stdlib. Secrets are never printed (prefixes only).
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import secrets
import subprocess
import sys
import tempfile
from pathlib import Path

PROD = os.environ.get("OPENPATHS_PROD_SSH", "administrator@93.127.141.100")
PROD_DIR = "/nvme0n1-disk/code/openpaths"
STATE = Path(__file__).with_name(".openpaths-rotation-state.json")
SSH = ["ssh", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=10"]

TOKEN_RE = re.compile(r"(?<![A-Za-z0-9_-])((?:sk-op-|op-)[A-Za-z0-9]{8,})")
SKIP_DIRS = {"node_modules", ".git", "target", "dist", "__pycache__", ".venv", "venv",
             "_work", "_diag", ".cache", "site-packages", "vendor", "test-results", "data", "datasets", "models",
             "checkpoints", "outputs", "build", ".next", "out", "tmp", "logs", "backups"}
MAX_DEPTH = 4
NAME_OK = re.compile(r"(^\.env|\.env($|\.)|\.envrc$|rc$|\.sh$|\.ya?ml$|\.toml$|\.conf$|\.ini$|\.service$)")
MAX_SIZE = 1_000_000
HEX64 = re.compile(r"^[0-9a-f]{64}$")
UUID = re.compile(r"^[0-9a-f-]{36}$")


def sha(key: str, mode: str = "hex") -> str:
    d = hashlib.sha256(key.encode()).digest()
    if mode == "b64url":  # app.nz hashToken: base64.RawURLEncoding(sha256)
        import base64
        return base64.urlsafe_b64encode(d).decode().rstrip("=")
    return d.hex()


# ---------------------------------------------------------------- agent (runs locally and on prod)
def iter_files(roots: list[str], extra: list[str]):
    seen = set()
    for f in extra:
        p = Path(f).expanduser()
        if p.is_file():
            seen.add(str(p))
            yield str(p)
    for root in roots:
        base = str(Path(root).expanduser()).rstrip("/").count("/")
        for d, dirs, files in os.walk(Path(root).expanduser()):
            dirs[:] = [x for x in dirs if x not in SKIP_DIRS and ".removed-" not in x]
            if d.count("/") - base >= MAX_DEPTH:
                dirs[:] = []
            for n in files:
                if not NAME_OK.search(n):
                    continue
                p = os.path.join(d, n)
                try:
                    if p in seen or os.path.islink(p) or os.path.getsize(p) > MAX_SIZE:
                        continue
                except OSError:
                    continue
                seen.add(p)
                yield p


def agent_scan(roots, extra, hashes, mode="hex", token_re=None):
    """-> {raw_key: [files]} for tokens whose sha256 is in `hashes`."""
    hs, found = set(hashes), {}
    rx = re.compile(token_re) if token_re else TOKEN_RE
    for p in iter_files(roots, extra):
        try:
            text = Path(p).read_text(errors="ignore")
        except OSError:
            continue
        for m in set(rx.findall(text)):
            if sha(m, mode) in hs:
                found.setdefault(m, []).append(p)
    return found


def agent_replace(mapping: dict[str, list[str]], repl: dict[str, str]):
    """mapping raw_old -> files; repl raw_old -> raw_new. Atomic in-place swap."""
    done = []
    for old, files in mapping.items():
        for p in files:
            path = Path(p)
            try:
                st = path.stat()
                text = path.read_text()
            except OSError as e:
                done.append([p, f"error: {e}"])
                continue
            if old not in text:
                done.append([p, "skip"])
                continue
            fd, tmp = tempfile.mkstemp(dir=path.parent)
            with os.fdopen(fd, "w") as fh:
                fh.write(text.replace(old, repl[old]))
            os.chmod(tmp, st.st_mode & 0o7777)
            try:
                os.chown(tmp, st.st_uid, st.st_gid)
            except PermissionError:
                pass
            os.replace(tmp, p)
            done.append([p, "ok"])
    return done


def agent_main():
    req = json.load(sys.stdin)
    if req["op"] == "scan":
        out = agent_scan(req["roots"], req["extra"], req["hashes"], req.get("mode", "hex"), req.get("token_re"))
    else:
        out = agent_replace(req["mapping"], req["repl"])
    json.dump(out, sys.stdout)


# ---------------------------------------------------------------- orchestrator
def local_roots():
    return ["/vfast/data/code", "/home/lee/code"], [str(Path.home() / n) for n in
            (".secretbashrc", ".bashrc", ".profile", ".zshrc", ".bash_profile", ".zshenv")]


def prod_roots():
    return ["/nvme0n1-disk/code", "/nvme0n1-disk/ci-runners", "/home/administrator/.config",
            "/etc/supervisor", "/etc/systemd/system", "/etc/default", "/opt"], \
           [f"/home/administrator/{n}" for n in (".secretbashrc", ".bashrc", ".profile", ".zshrc",
                                                  "start-claude-rc.sh")]


def run_agent(where: str, req: dict) -> dict:
    if where == "local":
        r = subprocess.run([sys.executable, "-I", __file__, "--agent"], input=json.dumps(req),
                           capture_output=True, text=True)
    else:
        remote = f"/tmp/.opk_agent_{secrets.token_hex(6)}.py"
        subprocess.run(SSH + [PROD, f"cat > {remote}"], input=Path(__file__).read_text(),
                       text=True, check=True)
        try:
            r = subprocess.run(SSH + [PROD, f"python3 -I {remote} --agent"], input=json.dumps(req),
                               capture_output=True, text=True)
        finally:
            subprocess.run(SSH + [PROD, f"rm -f {remote}"])
    if r.returncode:
        raise SystemExit(f"{where} agent failed: {r.stderr[-400:]}")
    return json.loads(r.stdout)


def psql(sql: str) -> str:
    """Run SQL on prod using the url from the service's config.yaml."""
    script = (f"cd {PROD_DIR}; u=$(grep -E '^\\s*url:' config.yaml | head -1 | "
              "sed -E 's/^\\s*url:\\s*\"?([^\"]*)\"?.*/\\1/'); psql \"$u\" -v ON_ERROR_STOP=1 -At -F '|'")
    r = subprocess.run(SSH + [PROD, script], input=sql, capture_output=True, text=True)
    if r.returncode:
        raise SystemExit("psql failed: " + "\n".join(l for l in r.stderr.splitlines() if "perl" not in l and "LANG" not in l and "LC_ALL" not in l)[-400:])
    return r.stdout


def active_keys(email: str) -> list[dict]:
    if not re.fullmatch(r"[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+", email):
        raise SystemExit("bad email")
    out = psql(f"select k.id, k.user_id, k.key_hash, k.key_prefix, replace(k.name,'|','/'), k.rate_limit_rpm, u.email "
               f"from api_keys k join users u on u.id=k.user_id where lower(u.email)=lower('{email}') "
               f"and not k.revoked order by k.created_at;")
    keys = []
    for line in out.splitlines():
        i, uid, h, pre, name, rpm, em = line.split("|")
        keys.append(dict(id=i, user_id=uid, hash=h, prefix=pre, name=name, rpm=int(rpm), email=em))
    return keys


def mint(user_id: str, name: str, rpm: int) -> tuple[str, str]:
    raw = "sk-op-" + secrets.token_hex(32)
    h, pre = sha(raw), raw[:14]
    assert UUID.match(user_id) and HEX64.match(h) and re.fullmatch(r"sk-op-[0-9a-f]{8}", pre)
    nm = name.replace("'", "''")
    out = psql(f"insert into api_keys (user_id, key_hash, key_prefix, name, rate_limit_rpm) "
               f"values ('{user_id}','{h}','{pre}','{nm}',{int(rpm)}) returning id;").strip().splitlines()[0]
    return raw, out


def revoke(ids: list[str]):
    ids = [i for i in ids if UUID.match(i)]
    if ids:
        quoted = ",".join(f"'{i}'" for i in ids)
        psql(f"update api_keys set revoked=true where id in ({quoted});")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--email", default="leepenkman@gmail.com")
    ap.add_argument("--apply", action="store_true")
    ap.add_argument("--finalize", action="store_true")
    ap.add_argument("--agent", action="store_true", help=argparse.SUPPRESS)
    a = ap.parse_args()
    if a.agent:
        return agent_main()

    if a.finalize:
        st = json.loads(STATE.read_text())
        ids = st["old_ids"] + st["unfound_ids"]
        quoted = ",".join(f"'{i}'" for i in ids if UUID.match(i))
        # a key used after --apply is still held by a running service: leave it alone
        busy = psql(f"select id, key_prefix, name from api_keys where id in ({quoted}) "
                    f"and last_used_at > '{st['applied_at']}'::timestamptz;").strip().splitlines()
        busy_ids = {l.split("|")[0] for l in busy}
        for l in busy:
            print("  still in use, NOT revoked:", l.split("|", 1)[1])
        revoke([i for i in ids if i not in busy_ids])
        print(f"revoked {len(ids) - len(busy_ids)} keys; {len(busy_ids)} skipped (in use). "
              f"Restart the services using them and re-run --finalize.")
        if busy_ids:
            st["old_ids"] = [i for i in st["old_ids"] if i in busy_ids]
            st["unfound_ids"] = [i for i in st["unfound_ids"] if i in busy_ids]
            STATE.write_text(json.dumps(st, indent=1))
            return
        STATE.rename(STATE.with_suffix(".done.json"))
        return

    keys = active_keys(a.email)
    byhash = {k["hash"]: k for k in keys}
    print(f"{len(keys)} active keys for {a.email}")
    found = {}  # raw -> {key, files: {where: [paths]}}
    for where, (roots, extra) in (("local", local_roots()), ("prod", prod_roots())):
        res = run_agent(where, dict(op="scan", roots=roots, extra=extra, hashes=list(byhash)))
        for raw, files in res.items():
            e = found.setdefault(raw, dict(key=byhash[sha(raw)], files={}))
            e["files"][where] = files
    for raw, e in found.items():
        k = e["key"]
        print(f"- {k['prefix']} '{k['name']}' rpm={k['rpm']} in: " +
              "; ".join(f"{w}:{len(f)} files" for w, f in e["files"].items()))
        for w, fs in e["files"].items():
            for f in fs:
                print(f"    {w} {f}")
    unfound = [k for k in keys if k["hash"] not in {sha(r) for r in found}]
    print(f"{len(unfound)} active keys not found in any file (will be revoked at --finalize):")
    for k in unfound:
        print(f"    {k['prefix']} '{k['name']}'")
    if not a.apply:
        print("\ndry run; use --apply")
        return

    repl, new_ids = {}, []
    for raw, e in found.items():
        k = e["key"]
        repl[raw], nid = mint(k["user_id"], k["name"], k["rpm"])
        new_ids.append(nid)
    for where in ("local", "prod"):
        mapping = {raw: e["files"][where] for raw, e in found.items() if where in e["files"]}
        for p, status in run_agent(where, dict(op="replace", mapping=mapping, repl=repl)):
            if status != "ok":
                print(f"  {where} {p}: {status}")
    pairs_file = Path.home() / "code/rotation/results/openpaths-key-pairs.json"
    if pairs_file.parent.is_dir():  # lets rotation/prod_sudo.py apply the same swaps to root-owned prod configs
        fd = os.open(pairs_file, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
        with os.fdopen(fd, "w") as fh:
            json.dump([[o, n] for o, n in repl.items()], fh)
    STATE.write_text(json.dumps(dict(
        applied_at=psql("select now();").strip(),
        old_ids=[e["key"]["id"] for e in found.values()], new_ids=new_ids,
        unfound_ids=[k["id"] for k in unfound],
        files=sorted({f for e in found.values() for fs in e["files"].values() for f in fs})), indent=1))
    os.chmod(STATE, 0o600)
    print(f"\nminted {len(repl)} new keys and swapped them in. Restart dependent services, "
          f"then run with --finalize to revoke old keys.")


if __name__ == "__main__":
    main()
