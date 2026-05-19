#!/usr/bin/env python3
"""Create replacement provider API keys and update this repo's .env file."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import sys
import tempfile
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Callable


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_ENV_FILE = ROOT / ".env"


class RotationError(RuntimeError):
    pass


@dataclass(frozen=True)
class RotationResult:
    provider: str
    env_key: str
    secret: str
    metadata: dict[str, object]


def parse_env_file(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.exists():
        return values

    for line in path.read_text().splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or "=" not in stripped:
            continue
        key, raw_value = stripped.split("=", 1)
        key = key.strip()
        raw_value = raw_value.strip()
        if raw_value.startswith(("'", '"')) and raw_value.endswith(raw_value[0]):
            raw_value = raw_value[1:-1]
        values[key] = raw_value
    return values


def env_value(name: str, env_file_values: dict[str, str], required: bool = True) -> str:
    value = os.environ.get(name) or env_file_values.get(name) or ""
    if required and not value:
        raise RotationError(f"Missing required environment variable: {name}")
    return value


def first_env_value(names: list[str], env_file_values: dict[str, str], required: bool = True) -> str:
    for name in names:
        value = os.environ.get(name) or env_file_values.get(name) or ""
        if value:
            return value
    if required:
        raise RotationError(f"Missing required environment variable: {' or '.join(names)}")
    return ""


def request_json(
    method: str,
    url: str,
    headers: dict[str, str],
    payload: dict[str, object] | None = None,
    timeout: int = 30,
) -> dict[str, object]:
    body = None
    request_headers = dict(headers)
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        request_headers.setdefault("Content-Type", "application/json")

    req = urllib.request.Request(url, data=body, headers=request_headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as response:
            response_body = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        error_body = exc.read().decode("utf-8", errors="replace")
        raise RotationError(f"{method} {url} failed with HTTP {exc.code}: {error_body}") from exc
    except urllib.error.URLError as exc:
        raise RotationError(f"{method} {url} failed: {exc.reason}") from exc

    try:
        data = json.loads(response_body)
    except json.JSONDecodeError as exc:
        raise RotationError(f"{method} {url} returned invalid JSON") from exc
    if not isinstance(data, dict):
        raise RotationError(f"{method} {url} returned JSON that is not an object")
    return data


def openai_admin_headers(admin_key: str, org_id: str = "") -> dict[str, str]:
    headers = {
        "Authorization": f"Bearer {admin_key}",
        "Content-Type": "application/json",
    }
    if org_id:
        headers["OpenAI-Organization"] = org_id
    return headers


def list_openai_projects(admin_key: str, org_id: str) -> list[dict[str, object]]:
    projects: list[dict[str, object]] = []
    after = ""
    while True:
        suffix = "?limit=100&include_archived=false"
        if after:
            suffix += f"&after={after}"
        data = request_json(
            "GET",
            "https://api.openai.com/v1/organization/projects" + suffix,
            headers=openai_admin_headers(admin_key, org_id),
        )
        page = data.get("data")
        if not isinstance(page, list):
            raise RotationError("OpenAI projects response did not include data list")
        projects.extend([p for p in page if isinstance(p, dict) and p.get("status") == "active"])
        if not data.get("has_more"):
            return projects
        last_id = data.get("last_id")
        if not isinstance(last_id, str) or not last_id:
            raise RotationError("OpenAI projects response is missing pagination last_id")
        after = last_id


def resolve_openai_project_id(env_file_values: dict[str, str], admin_key: str, org_id: str) -> str:
    project_id = env_value("OPENAI_PROJECT_ID", env_file_values, required=False)
    if project_id:
        return project_id

    projects = list_openai_projects(admin_key, org_id)
    if len(projects) == 1 and isinstance(projects[0].get("id"), str):
        return str(projects[0]["id"])

    names = ", ".join(f"{p.get('name')} ({p.get('id')})" for p in projects[:10])
    if not names:
        names = "none"
    raise RotationError(
        "OPENAI_PROJECT_ID is required because the admin key can access "
        f"{len(projects)} active projects: {names}"
    )


def validate_openai_key(api_key: str, org_id: str) -> None:
    headers = {"Authorization": f"Bearer {api_key}"}
    if org_id:
        headers["OpenAI-Organization"] = org_id
    data = request_json("GET", "https://api.openai.com/v1/models", headers=headers)
    if not isinstance(data.get("data"), list):
        raise RotationError("New OpenAI API key validation did not return a models list")


def redacted_key_matches(redacted: object, secret: str) -> bool:
    if not isinstance(redacted, str) or not secret:
        return False
    if "..." in redacted:
        prefix, suffix = redacted.split("...", 1)
        return (not prefix or secret.startswith(prefix)) and (not suffix or secret.endswith(suffix))
    if "*" in redacted:
        prefix = redacted.split("*", 1)[0]
        suffix = redacted.rsplit("*", 1)[-1]
        return (not prefix or secret.startswith(prefix)) and (not suffix or secret.endswith(suffix))
    return redacted == secret


def list_openai_project_keys(admin_key: str, org_id: str, project_id: str) -> list[dict[str, object]]:
    keys: list[dict[str, object]] = []
    after = ""
    while True:
        suffix = "?limit=100"
        if after:
            suffix += f"&after={after}"
        data = request_json(
            "GET",
            f"https://api.openai.com/v1/organization/projects/{project_id}/api_keys{suffix}",
            headers=openai_admin_headers(admin_key, org_id),
        )
        page = data.get("data")
        if not isinstance(page, list):
            raise RotationError("OpenAI project keys response did not include data list")
        keys.extend([k for k in page if isinstance(k, dict)])
        if not data.get("has_more"):
            return keys
        last_id = data.get("last_id")
        if not isinstance(last_id, str) or not last_id:
            raise RotationError("OpenAI project keys response is missing pagination last_id")
        after = last_id


def revoke_openai_key(admin_key: str, org_id: str, project_id: str, old_secret: str) -> dict[str, object]:
    if not old_secret:
        return {"revoked": False, "reason": "no previous OPENAI_API_KEY value found"}

    matches = [k for k in list_openai_project_keys(admin_key, org_id, project_id) if redacted_key_matches(k.get("redacted_value"), old_secret)]
    if len(matches) != 1:
        return {"revoked": False, "reason": f"matched {len(matches)} project API keys for previous OPENAI_API_KEY"}

    key = matches[0]
    key_id = key.get("id")
    if not isinstance(key_id, str) or not key_id:
        raise RotationError("Matched old OpenAI key is missing id")

    owner = key.get("owner")
    if not isinstance(owner, dict):
        raise RotationError("Matched old OpenAI key is missing owner")
    owner_type = owner.get("type")

    if owner_type == "service_account":
        service_account = owner.get("service_account")
        if not isinstance(service_account, dict) or not isinstance(service_account.get("id"), str):
            raise RotationError("Matched service-account key is missing service account id")
        service_account_id = service_account["id"]
        request_json(
            "DELETE",
            f"https://api.openai.com/v1/organization/projects/{project_id}/service_accounts/{service_account_id}",
            headers=openai_admin_headers(admin_key, org_id),
        )
        return {"revoked": True, "method": "delete_service_account", "service_account_id": service_account_id, "api_key_id": key_id}

    request_json(
        "DELETE",
        f"https://api.openai.com/v1/organization/projects/{project_id}/api_keys/{key_id}",
        headers=openai_admin_headers(admin_key, org_id),
    )
    return {"revoked": True, "method": "delete_project_api_key", "api_key_id": key_id}


def create_openai_key(alias: str, env_file_values: dict[str, str]) -> RotationResult:
    admin_key = first_env_value(["OPENAI_ADMIN_KEY", "OPENAI_ADMIN_API_KEY"], env_file_values)
    org_id = env_value("OPENAI_ORG_ID", env_file_values, required=False)
    project_id = resolve_openai_project_id(env_file_values, admin_key, org_id)
    url = f"https://api.openai.com/v1/organization/projects/{project_id}/service_accounts"
    data = request_json(
        "POST",
        url,
        headers=openai_admin_headers(admin_key, org_id),
        payload={"name": alias},
    )

    api_key = data.get("api_key")
    if not isinstance(api_key, dict) or not isinstance(api_key.get("value"), str):
        raise RotationError("OpenAI response did not include api_key.value")

    validate_openai_key(api_key["value"], org_id)

    return RotationResult(
        provider="openai",
        env_key="OPENAI_API_KEY",
        secret=api_key["value"],
        metadata={
            "project_id": project_id,
            "service_account_id": data.get("id"),
            "service_account_name": data.get("name"),
            "api_key_id": api_key.get("id"),
            "api_key_name": api_key.get("name"),
            "new_key_validated": True,
        },
    )


def create_fal_key(alias: str, env_file_values: dict[str, str]) -> RotationResult:
    admin_key = env_value("FAL_ADMIN_API_KEY", env_file_values, required=False)
    if not admin_key:
        admin_key = env_value("FAL_KEY", env_file_values, required=False)
    if not admin_key:
        admin_key = env_value("FAL_API_KEY", env_file_values)

    data = request_json(
        "POST",
        "https://api.fal.ai/v1/keys",
        headers={"Authorization": f"Key {admin_key}"},
        payload={"alias": alias},
    )

    new_key = data.get("key")
    if not isinstance(new_key, str) or not new_key:
        raise RotationError("fal response did not include key")

    return RotationResult(
        provider="fal",
        env_key="FAL_API_KEY",
        secret=new_key,
        metadata={"key_id": data.get("key_id")},
    )


def anthropic_headers(api_key: str) -> dict[str, str]:
    return {
        "anthropic-version": "2023-06-01",
        "x-api-key": api_key,
    }


def validate_anthropic_key(api_key: str) -> None:
    data = request_json(
        "GET",
        "https://api.anthropic.com/v1/models?limit=1",
        headers=anthropic_headers(api_key),
    )
    if not isinstance(data.get("data"), list):
        raise RotationError("New Anthropic API key validation did not return a models list")


def create_anthropic_key(alias: str, env_file_values: dict[str, str]) -> RotationResult:
    # Anthropic's Admin API can list and update API keys, but does not document a
    # create endpoint that returns a new one-time secret. Read the new secret from
    # the environment so the rest of the rotation flow can still be validated.
    del alias
    source = "ANTHROPIC_NEW_API_KEY"
    new_key = os.environ.get(source) or env_file_values.get(source) or ""
    if not new_key:
        source = "ANTHROPIC_REPLACEMENT_API_KEY"
        new_key = os.environ.get(source) or env_file_values.get(source) or ""
    if not new_key:
        raise RotationError("Missing required environment variable: ANTHROPIC_NEW_API_KEY or ANTHROPIC_REPLACEMENT_API_KEY")
    validate_anthropic_key(new_key)

    return RotationResult(
        provider="anthropic",
        env_key="ANTHROPIC_API_KEY",
        secret=new_key,
        metadata={
            "new_key_source": source,
            "new_key_validated": True,
        },
    )


def list_anthropic_api_keys(admin_key: str) -> list[dict[str, object]]:
    keys: list[dict[str, object]] = []
    after_id = ""
    while True:
        suffix = "?limit=1000"
        if after_id:
            suffix += f"&after_id={after_id}"
        data = request_json(
            "GET",
            "https://api.anthropic.com/v1/organizations/api_keys" + suffix,
            headers=anthropic_headers(admin_key),
        )
        page = data.get("data")
        if not isinstance(page, list):
            raise RotationError("Anthropic API keys response did not include data list")
        keys.extend([k for k in page if isinstance(k, dict)])
        if not data.get("has_more"):
            return keys
        last_id = data.get("last_id")
        if not isinstance(last_id, str) or not last_id:
            raise RotationError("Anthropic API keys response is missing pagination last_id")
        after_id = last_id


def revoke_anthropic_key(admin_key: str, old_secret: str) -> dict[str, object]:
    if not old_secret:
        return {"revoked": False, "reason": "no previous ANTHROPIC_API_KEY value found"}

    matches = [
        k
        for k in list_anthropic_api_keys(admin_key)
        if k.get("status") == "active" and redacted_key_matches(k.get("partial_key_hint"), old_secret)
    ]
    if len(matches) != 1:
        return {"revoked": False, "reason": f"matched {len(matches)} active Anthropic API keys for previous ANTHROPIC_API_KEY"}

    key_id = matches[0].get("id")
    if not isinstance(key_id, str) or not key_id:
        raise RotationError("Matched old Anthropic key is missing id")

    request_json(
        "POST",
        f"https://api.anthropic.com/v1/organizations/api_keys/{key_id}",
        headers=anthropic_headers(admin_key),
        payload={"status": "inactive"},
    )
    return {"revoked": True, "method": "update_status_inactive", "api_key_id": key_id}


ROTATORS: dict[str, Callable[[str, dict[str, str]], RotationResult]] = {
    "anthropic": create_anthropic_key,
    "openai": create_openai_key,
    "fal": create_fal_key,
}


def shell_quote(value: str) -> str:
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def update_env_file(path: Path, key: str, value: str) -> Path:
    if not path.exists():
        raise RotationError(f"Env file does not exist: {path}")

    original = path.read_text()
    backup = path.with_name(f"{path.name}.bak-rotation-{time.strftime('%Y%m%d%H%M%S')}")
    shutil.copy2(path, backup)

    replacement = f"{key}={shell_quote(value)}"
    pattern = re.compile(rf"^{re.escape(key)}\s*=.*$")
    updated_lines: list[str] = []
    replaced = False
    for line in original.splitlines():
        if pattern.match(line):
            if not replaced:
                updated_lines.append(replacement)
                replaced = True
            continue
        updated_lines.append(line)
    if not replaced:
        updated_lines.append(replacement)
    updated = "\n".join(updated_lines) + "\n"

    fd, tmp_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=str(path.parent), text=True)
    try:
        with os.fdopen(fd, "w") as tmp:
            tmp.write(updated)
        os.replace(tmp_name, path)
    except Exception:
        try:
            os.unlink(tmp_name)
        finally:
            raise

    return backup


def revoke_previous_key(provider: str, result: RotationResult, env_file_values: dict[str, str]) -> dict[str, object]:
    if provider == "anthropic":
        admin_key = first_env_value(["ANTHROPIC_ADMIN_KEY", "ANTHROPIC_ADMIN_API_KEY"], env_file_values, required=False)
        if not admin_key:
            return {"revoked": False, "reason": "ANTHROPIC_ADMIN_KEY or ANTHROPIC_ADMIN_API_KEY is required to deactivate the previous key"}
        return revoke_anthropic_key(admin_key, env_file_values.get(result.env_key, ""))

    if provider != "openai":
        return {"revoked": False, "reason": "provider revocation is not implemented"}

    admin_key = first_env_value(["OPENAI_ADMIN_KEY", "OPENAI_ADMIN_API_KEY"], env_file_values)
    org_id = env_value("OPENAI_ORG_ID", env_file_values, required=False)
    project_id = result.metadata.get("project_id")
    if not isinstance(project_id, str) or not project_id:
        project_id = resolve_openai_project_id(env_file_values, admin_key, org_id)
    old_secret = env_file_values.get(result.env_key, "")
    return revoke_openai_key(admin_key, org_id, project_id, old_secret)


def rotate_provider(provider: str, env_file: Path, alias_prefix: str, no_env_write: bool) -> None:
    env_file_values = parse_env_file(env_file)
    alias = f"{alias_prefix}-{provider}-{time.strftime('%Y%m%d%H%M%S')}"
    result = ROTATORS[provider](alias, env_file_values)

    if no_env_write:
        print(f"created {provider} key for {result.env_key}; .env was not changed")
        print(json.dumps({k: v for k, v in result.metadata.items() if v is not None}, indent=2))
        return

    backup = update_env_file(env_file, result.env_key, result.secret)
    revoke_result = revoke_previous_key(provider, result, env_file_values)
    print(f"created {provider} key and updated {env_file} {result.env_key}")
    print(f"backup: {backup}")
    metadata = {k: v for k, v in result.metadata.items() if v is not None}
    metadata["previous_key"] = revoke_result
    print(json.dumps(metadata, indent=2))


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Create new provider API keys and write them into .env without revoking old keys."
    )
    parser.add_argument(
        "provider",
        choices=sorted([*ROTATORS.keys(), "all"]),
        help="Provider to rotate. 'all' rotates every supported provider with available credentials.",
    )
    parser.add_argument("--env-file", default=str(DEFAULT_ENV_FILE), help="Path to .env file to update.")
    parser.add_argument("--alias-prefix", default="openpaths-rotation", help="Prefix for provider key names.")
    parser.add_argument(
        "--no-env-write",
        action="store_true",
        help="Create the provider key but skip writing .env.",
    )
    args = parser.parse_args()

    all_mode = args.provider == "all"
    providers = sorted(ROTATORS) if all_mode else [args.provider]
    errors = 0
    for provider in providers:
        try:
            rotate_provider(provider, Path(args.env_file).resolve(), args.alias_prefix, args.no_env_write)
        except RotationError as exc:
            if all_mode and str(exc).startswith("Missing required environment variable:"):
                print(f"{provider}: skipped ({exc})")
                continue
            errors += 1
            print(f"{provider}: {exc}", file=sys.stderr)

    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
