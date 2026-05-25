#!/usr/bin/env python3
"""Build the static ZImage art index consumed by /art.

The intended production flow is deliberately resumable:

  python scripts/migrate_zimage_art_index.py \
    --source /sdb-disk/cutedsl-images/prompts.jsonl \
    --limit 40000 \
    --regenerate \
    --upload

Input may be JSONL, JSON array, or CSV. Rows can be prompt-only, or can carry
existing image_url/thumb_url fields. With --regenerate, missing images are
generated through OpenPaths /v1/images/generations using model=zimage, steps=20,
then uploaded to the configured R2 bucket. The script writes chunked JSON data
and a manifest under static/data/zimage-art so the frontend can browse without
shipping a 40k-record bundle.
"""

from __future__ import annotations

import argparse
import csv
import hashlib
import io
import json
import os
import random
import re
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable

import boto3
import requests
from PIL import Image


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_PUBLIC_BASE = "https://openpathsstatic.openpaths.io"
DEFAULT_DATA_PREFIX = "static/data/zimage-art"
DEFAULT_IMAGE_PREFIX = "static/uploads/zimage-art"
DEFAULT_OUTPUT_DIR = ROOT / "tmp" / "zimage-art-index"

BLOCKED_RE = re.compile(
    r"\b("
    r"nsfw|nude|nudity|naked|topless|bottomless|porn|pornographic|xxx|sex|sexual|erotic|"
    r"hentai|loli|shota|lolicon|shotacon|child|children|underage|minor|teen|schoolgirl|"
    r"schoolboy|baby|infant|toddler|gore|torture|rape|suicide|self[- ]harm|nazi|hitler"
    r")\b",
    re.IGNORECASE,
)


@dataclass
class SourceRecord:
    prompt: str
    image_url: str = ""
    thumb_url: str = ""
    width: int = 1024
    height: int = 1024
    seed: int = 0
    steps: int = 20
    source: str = "migration"


def load_env_files() -> None:
    for path in (ROOT / ".env", Path.home() / ".secretbashrc"):
        if not path.exists():
            continue
        for raw in path.read_text().splitlines():
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            if line.startswith("export "):
                line = line[len("export ") :]
            if "=" not in line:
                continue
            key, value = line.split("=", 1)
            os.environ.setdefault(key.strip(), value.strip().strip("'\""))


def clean_prompt(prompt: str) -> str:
    prompt = " ".join((prompt or "").strip().split())
    if len(prompt) < 12 or len(prompt) > 520:
        return ""
    if prompt.startswith("http://") or prompt.startswith("https://"):
        return ""
    if BLOCKED_RE.search(prompt):
        return ""
    return prompt


def slugify(value: str, max_len: int = 72) -> str:
    value = re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")
    return value[:max_len].strip("-") or "zimage-art"


def stable_id(prompt: str) -> str:
    return hashlib.sha256(prompt.lower().encode("utf-8")).hexdigest()[:20]


def tags_for_prompt(prompt: str, limit: int = 10) -> list[str]:
    stop = {"with", "from", "into", "that", "this", "style", "image", "art", "very", "high", "quality"}
    tags: list[str] = []
    seen: set[str] = set()
    for word in re.findall(r"[a-zA-Z0-9]{4,}", prompt.lower()):
        if word in stop or word in seen:
            continue
        seen.add(word)
        tags.append(word)
        if len(tags) >= limit:
            break
    return tags


def read_source(path: Path) -> Iterable[SourceRecord]:
    suffix = path.suffix.lower()
    if suffix == ".jsonl":
        with path.open() as fh:
            for line in fh:
                if line.strip():
                    yield record_from_mapping(json.loads(line), str(path))
        return
    if suffix == ".json":
        data = json.loads(path.read_text())
        if isinstance(data, dict):
            data = data.get("items") or data.get("images") or data.get("data") or []
        for row in data:
            yield record_from_mapping(row, str(path))
        return
    if suffix == ".csv":
        with path.open(newline="") as fh:
            for row in csv.DictReader(fh):
                yield record_from_mapping(row, str(path))
        return
    raise SystemExit(f"Unsupported source format: {path}")


def record_from_mapping(row: dict[str, Any], source: str) -> SourceRecord:
    prompt = clean_prompt(str(row.get("prompt") or row.get("text") or row.get("caption") or ""))
    if not prompt:
        return SourceRecord(prompt="")
    image_url = str(row.get("imageUrl") or row.get("image_url") or row.get("url") or row.get("file_path") or "")
    thumb_url = str(row.get("thumbUrl") or row.get("thumb_url") or row.get("thumbnail") or row.get("med_path") or "")
    return SourceRecord(
        prompt=prompt,
        image_url=image_url,
        thumb_url=thumb_url,
        width=int(row.get("width") or 1024),
        height=int(row.get("height") or 1024),
        seed=int(row.get("seed") or 0),
        steps=int(row.get("steps") or 20),
        source=source,
    )


def sample_records(source: Path, limit: int, seed: int) -> list[SourceRecord]:
    seen: set[str] = set()
    records: list[SourceRecord] = []
    for record in read_source(source):
        if not record.prompt:
            continue
        key = record.prompt.lower()
        if key in seen:
            continue
        seen.add(key)
        records.append(record)
    rng = random.Random(seed)
    rng.shuffle(records)
    return records[:limit]


def openpaths_headers() -> dict[str, str]:
    key = os.environ.get("OPENPATHS_API_KEY") or os.environ.get("TEST_API_KEY")
    if not key:
        raise RuntimeError("OPENPATHS_API_KEY is required when --regenerate is set")
    return {"Authorization": f"Bearer {key}", "Content-Type": "application/json"}


def generate_zimage(prompt: str, steps: int, base_url: str) -> str:
    payload = {
        "model": "zimage",
        "prompt": prompt,
        "size": "1024x1024",
        "n": 1,
        "steps": steps,
        "num_inference_steps": steps,
        "response_format": "url",
    }
    resp = requests.post(
        f"{base_url.rstrip('/')}/v1/images/generations",
        headers=openpaths_headers(),
        json=payload,
        timeout=300,
    )
    resp.raise_for_status()
    data = resp.json()
    images = data.get("data") or []
    if not images:
        raise RuntimeError(f"image generation returned no data: {data}")
    image_url = images[0].get("url") or images[0].get("image_url")
    if not image_url:
        raise RuntimeError(f"image generation returned no URL: {data}")
    return image_url


def download_image(url: str) -> bytes:
    resp = requests.get(url, timeout=300)
    resp.raise_for_status()
    return resp.content


def reencode_webp(raw: bytes, quality: int) -> bytes:
    with Image.open(io.BytesIO(raw)) as image:
        out = io.BytesIO()
        image.convert("RGB").save(out, format="WEBP", quality=quality, method=6)
        return out.getvalue()


def r2_client():
    endpoint = os.environ.get("R2_ENDPOINT")
    access_key = os.environ.get("R2_ACCESS_KEY") or os.environ.get("CLOUDFLARE_R2_ACCESS_KEY_ID")
    secret_key = os.environ.get("R2_SECRET_KEY") or os.environ.get("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
    if not endpoint or not access_key or not secret_key:
        raise RuntimeError("R2_ENDPOINT, R2_ACCESS_KEY, and R2_SECRET_KEY are required for --upload")
    return boto3.client(
        "s3",
        endpoint_url=endpoint,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        region_name="auto",
    )


def upload_bytes(client, bucket: str, key: str, body: bytes, content_type: str) -> str:
    client.put_object(
        Bucket=bucket,
        Key=key,
        Body=body,
        ContentType=content_type,
        CacheControl="public, max-age=31536000, immutable" if content_type.startswith("image/") else "public, max-age=300",
    )
    public_base = os.environ.get("R2_PUBLIC_URL") or DEFAULT_PUBLIC_BASE
    return f"{public_base.rstrip('/')}/{key}"


def build_item(record: SourceRecord, image_url: str, thumb_url: str, steps: int) -> dict[str, Any]:
    item_id = stable_id(record.prompt)
    return {
        "id": item_id,
        "slug": f"{slugify(record.prompt)}-{item_id[:8]}",
        "title": "ZImage generated artwork",
        "prompt": record.prompt,
        "imageUrl": image_url,
        "thumbUrl": thumb_url or image_url,
        "width": record.width,
        "height": record.height,
        "model": "zimage",
        "seed": record.seed,
        "steps": steps,
        "source": record.source,
        "tags": tags_for_prompt(record.prompt),
    }


def process_record(record: SourceRecord, args, client, bucket: str | None) -> dict[str, Any]:
    image_url = record.image_url
    if args.regenerate:
        image_url = generate_zimage(record.prompt, args.steps, args.openpaths_base_url)
    if args.upload_images and image_url:
        raw = download_image(image_url)
        webp = reencode_webp(raw, args.webp_quality)
        key = f"{args.image_prefix.rstrip('/')}/{stable_id(record.prompt)}.webp"
        if args.upload:
            image_url = upload_bytes(client, bucket or "", key, webp, "image/webp")
        else:
            local_path = Path(args.output_dir) / key
            local_path.parent.mkdir(parents=True, exist_ok=True)
            local_path.write_bytes(webp)
            image_url = str(local_path)
    return build_item(record, image_url, record.thumb_url, args.steps)


def write_index(items: list[dict[str, Any]], args, client, bucket: str | None) -> None:
    out_dir = Path(args.output_dir)
    chunk_dir = out_dir / "chunks"
    chunk_dir.mkdir(parents=True, exist_ok=True)

    chunks = []
    for idx in range(0, len(items), args.chunk_size):
        batch = items[idx:idx + args.chunk_size]
        chunk_name = f"chunks/chunk-{idx // args.chunk_size:04d}.json"
        body = json.dumps(batch, separators=(",", ":")).encode("utf-8")
        (out_dir / chunk_name).write_bytes(body)
        key = f"{args.data_prefix.rstrip('/')}/{chunk_name}"
        if args.upload:
            upload_bytes(client, bucket or "", key, body, "application/json; charset=utf-8")
        chunks.append({"path": chunk_name, "count": len(batch)})

    manifest = {
        "version": 1,
        "kind": "zimage-art-index",
        "model": "zimage",
        "count": len(items),
        "generatedCount": sum(1 for item in items if item.get("imageUrl")),
        "publicBase": f"{(os.environ.get('R2_PUBLIC_URL') or DEFAULT_PUBLIC_BASE).rstrip('/')}/{args.data_prefix.rstrip('/')}",
        "generatedAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "chunks": chunks,
    }
    body = json.dumps(manifest, indent=2).encode("utf-8")
    (out_dir / "manifest.json").write_bytes(body)
    if args.upload:
        upload_bytes(client, bucket or "", f"{args.data_prefix.rstrip('/')}/manifest.json", body, "application/json; charset=utf-8")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Create the OpenPaths static ZImage art index.")
    parser.add_argument("--source", required=True, help="JSONL, JSON, or CSV prompt/image export")
    parser.add_argument("--limit", type=int, default=40000)
    parser.add_argument("--sample-seed", type=int, default=42)
    parser.add_argument("--steps", type=int, default=20)
    parser.add_argument("--chunk-size", type=int, default=5000)
    parser.add_argument("--concurrency", type=int, default=2)
    parser.add_argument("--output-dir", default=str(DEFAULT_OUTPUT_DIR))
    parser.add_argument("--resume-file", default="", help="JSONL file of completed items; defaults to <output-dir>/items.jsonl")
    parser.add_argument("--data-prefix", default=DEFAULT_DATA_PREFIX)
    parser.add_argument("--image-prefix", default=DEFAULT_IMAGE_PREFIX)
    parser.add_argument("--openpaths-base-url", default=os.environ.get("OPENPATHS_BASE_URL", "https://openpaths.io"))
    parser.add_argument("--webp-quality", type=int, default=85)
    parser.add_argument("--regenerate", action="store_true", help="Regenerate every selected prompt through zimage")
    parser.add_argument("--upload-images", action="store_true", help="Download/re-encode/upload image files")
    parser.add_argument("--upload", action="store_true", help="Upload manifest/chunks/images to R2")
    return parser


def main() -> int:
    load_env_files()
    args = build_parser().parse_args()
    source = Path(args.source)
    if not source.exists():
        raise SystemExit(f"Source file not found: {source}")
    if args.regenerate:
        args.upload_images = True

    records = sample_records(source, args.limit, args.sample_seed)
    resume_path = Path(args.resume_file) if args.resume_file else Path(args.output_dir) / "items.jsonl"
    resume_path.parent.mkdir(parents=True, exist_ok=True)

    existing: dict[str, dict[str, Any]] = {}
    if resume_path.exists():
        with resume_path.open() as fh:
            for line in fh:
                if not line.strip():
                    continue
                item = json.loads(line)
                if item.get("id"):
                    existing[item["id"]] = item

    pending = [record for record in records if stable_id(record.prompt) not in existing]
    print(
        f"Selected {len(records):,} safe unique prompts from {source}; "
        f"{len(existing):,} resumed, {len(pending):,} pending",
        file=sys.stderr,
    )

    client = r2_client() if args.upload else None
    bucket = os.environ.get("R2_BUCKET") or os.environ.get("CLOUDFLARE_R2_BUCKET") or "openpathsstatic"

    items: list[dict[str, Any]] = list(existing.values())
    with resume_path.open("a") as resume_fh, ThreadPoolExecutor(max_workers=max(1, args.concurrency)) as pool:
        futures = [pool.submit(process_record, record, args, client, bucket) for record in pending]
        for i, future in enumerate(as_completed(futures), start=1):
            try:
                item = future.result()
                items.append(item)
                resume_fh.write(json.dumps(item, separators=(",", ":")) + "\n")
                resume_fh.flush()
            except Exception as exc:
                print(f"WARN: skipped record: {exc}", file=sys.stderr)
            if i % 250 == 0:
                print(f"Processed {i:,}/{len(pending):,}; kept {len(items):,}", file=sys.stderr)

    items.sort(key=lambda item: item["id"])
    write_index(items, args, client, bucket)
    print(f"Wrote ZImage art index with {len(items):,} items to {args.output_dir}", file=sys.stderr)
    if args.upload:
        print(f"Published {(os.environ.get('R2_PUBLIC_URL') or DEFAULT_PUBLIC_BASE).rstrip('/')}/{args.data_prefix.rstrip('/')}/manifest.json")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
