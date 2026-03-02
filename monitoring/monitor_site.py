#!/usr/bin/env python3
import time
import subprocess
import urllib.request
import json
import ssl
from urllib.error import URLError, HTTPError
from datetime import datetime, timezone
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
LOG_FILE = SCRIPT_DIR / "logs" / "monitor.log"
CLAUDE_FIX_LOG = SCRIPT_DIR / "logs" / "claude_fix.log"
PROJECT_DIR = SCRIPT_DIR.parent

SITES = [
    "https://openpaths.io",
]

API_ENDPOINTS = [
    {
        "url": "https://openpaths.io/v1/models",
        "method": "GET",
        "headers": {"Accept": "application/json"},
        "name": "models_api",
    },
]

SSH_PASS = "ka3iMI4OSNvgFcREuDaQyLguFxuP"
API_HOST = "administrator@93.127.141.100"
REMOTE_DIR = "/nvme0n1-disk/code/openpaths"

CLAUDE_FIX_PROMPT = """Site {url} is DOWN. Investigate and fix.

Info:
- Server: administrator@93.127.141.100
- SSH pass: {ssh_pass}
- Check: sshpass -p '{ssh_pass}' ssh -o StrictHostKeyChecking=no {host} 'sudo supervisorctl status openpaths'
- Restart: sshpass -p '{ssh_pass}' ssh -o StrictHostKeyChecking=no {host} 'sudo supervisorctl restart openpaths'
- Logs: sshpass -p '{ssh_pass}' ssh -o StrictHostKeyChecking=no {host} 'tail -50 /var/log/supervisor/openpaths-error.log'
- Binary: {remote_dir}/openpaths-api
- Config: {remote_dir}/config.yaml
- Deploy: bash deploy.sh api
- Test after fix: curl -s https://openpaths.io/v1/models

Fix the issue then verify site is back up."""


def log(message):
    LOG_FILE.parent.mkdir(parents=True, exist_ok=True)
    with open(LOG_FILE, "a") as fh:
        fh.write(f"{datetime.now(timezone.utc).isoformat()} {message}\n")


def check_site(url, timeout=15):
    try:
        ctx = ssl.create_default_context()
        req = urllib.request.Request(url, headers={
            "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
        })
        with urllib.request.urlopen(req, timeout=timeout, context=ctx) as resp:
            return resp.status < 400
    except (HTTPError, URLError, ValueError, ssl.SSLError):
        return False


def check_api_endpoint(endpoint, timeout=15):
    try:
        ctx = ssl.create_default_context()
        data = json.dumps(endpoint["body"]).encode() if endpoint.get("body") else None
        req = urllib.request.Request(
            endpoint["url"], data=data,
            headers=endpoint.get("headers", {}),
            method=endpoint.get("method", "GET"),
        )
        with urllib.request.urlopen(req, timeout=timeout, context=ctx) as resp:
            body = resp.read().decode()
            info = body[:200] + "..." if len(body) > 200 else body
            return resp.status < 400, f"Status: {resp.status}, Response: {info}"
    except (HTTPError, URLError, ValueError) as e:
        return False, f"Error: {e}"


def check_js_errors(url="https://openpaths.io", timeout=30):
    """Use playwright to check for JS errors on homepage."""
    try:
        result = subprocess.run(
            ["node", str(SCRIPT_DIR / "check_js.mjs"), url],
            capture_output=True, text=True, timeout=timeout,
            cwd=str(PROJECT_DIR),
        )
        if result.returncode != 0:
            errors = result.stdout.strip() or result.stderr.strip()
            log(f"JS errors on {url}: {errors}")
            return False, errors
        log(f"JS check {url}: clean")
        return True, ""
    except (subprocess.TimeoutExpired, FileNotFoundError) as e:
        log(f"JS check skipped: {e}")
        return True, ""


def notify_down(url):
    subprocess.run(["espeak", f"{url} is down"], check=False,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    log(f"Spawning Claude to fix {url}")
    prompt = CLAUDE_FIX_PROMPT.format(
        url=url, ssh_pass=SSH_PASS, host=API_HOST, remote_dir=REMOTE_DIR,
    )
    CLAUDE_FIX_LOG.parent.mkdir(parents=True, exist_ok=True)
    subprocess.Popen(
        ["claude", "--dangerously-skip-permissions", "-p", prompt],
        cwd=str(PROJECT_DIR),
        stdout=open(CLAUDE_FIX_LOG, "a"),
        stderr=subprocess.STDOUT,
    )


def monitor(interval=60, iterations=None, check_js=True):
    statuses = {}
    count = 0
    while iterations is None or count < iterations:
        for site in SITES:
            ok = check_site(site)
            log(f"{site} {'up' if ok else 'down'}")
            prev = statuses.get(site, True)
            if not ok and prev:
                notify_down(site)
            statuses[site] = ok

        for ep in API_ENDPOINTS:
            name = ep.get("name", ep["url"])
            ok, info = check_api_endpoint(ep)
            log(f"API {name} {'up' if ok else 'down'} - {info}")
            prev = statuses.get(name, True)
            if not ok and prev:
                notify_down(f"API {name}")
            statuses[name] = ok

        if check_js and count % 5 == 0:
            ok, errors = check_js_errors()
            if not ok:
                prev = statuses.get("js_check", True)
                if prev:
                    notify_down("JS errors on https://openpaths.io")
                statuses["js_check"] = False
            else:
                statuses["js_check"] = True

        count += 1
        time.sleep(interval)


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--interval", type=int, default=60)
    parser.add_argument("--iterations", type=int, default=None)
    parser.add_argument("--no-js", action="store_true")
    args = parser.parse_args()
    monitor(interval=args.interval, iterations=args.iterations, check_js=not args.no_js)
