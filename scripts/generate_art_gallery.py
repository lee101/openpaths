#!/usr/bin/env python3
"""Generate and upload landing-page art gallery assets.

This script:
- loads credentials from `.env`, `../netwrck/.env`, and `~/.secretbashrc`
- calls configured image providers
- downloads the result
- converts it to WebP at quality 85
- uploads it to the OpenPaths static bucket

The script is intentionally provider-aware so we can add new batches
without hand-editing the landing page whenever another key comes online.
"""

from __future__ import annotations

import argparse
import io
import json
import os
import re
import sys
import textwrap
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import boto3
import requests
from PIL import Image


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OUTPUT_DIR = ROOT / "tmp" / "art-gallery"
DEFAULT_PREFIX = "static/uploads/landing/art-playground"
DEFAULT_PUBLIC_BASE = "https://openpathsstatic.openpaths.io"


PROMPTS: dict[str, dict[str, str]] = {
    "archive-chapel": {
        "title": "Archive Chapel",
        "prompt": (
            "A brutalist chapel grown from translucent salt crystals on the edge of a frozen black lake, "
            "thin dawn light, long shadows, hyper-detailed editorial architecture photography, no people"
        ),
    },
    "gravity-orchard": {
        "title": "Gravity Orchard",
        "prompt": (
            "An orchard where every fruit is a tiny moon with its own orbit, farmers on ladders harvesting by "
            "magnetic lantern light, surreal realism, cinematic depth, atmospheric fog"
        ),
    },
    "paper-nautilus": {
        "title": "Paper Nautilus",
        "prompt": (
            "A giant nautilus made of folded maps and train tickets swimming through a dry museum atrium, "
            "sunbeams, dust particles, whimsical but physically grounded, museum-grade photography"
        ),
    },
    "ocean-typewriter": {
        "title": "Ocean Typewriter",
        "prompt": (
            "A vintage typewriter resting on the seafloor, each keypress releasing schools of silver fish shaped "
            "like punctuation marks, teal water, volumetric light, dreamlike macro photography"
        ),
    },
    "clockmaker-desert": {
        "title": "Clockmaker Desert",
        "prompt": (
            "A desert workshop where watch gears are half-buried like fossils and a lone mechanic tunes time with "
            "copper tools, warm dusk palette, tactile realism, intricate details"
        ),
    },
    "monsoon-teahouse": {
        "title": "Monsoon Teahouse",
        "prompt": (
            "A floating teahouse drifting through a tropical monsoon above a neon city, rain slanting sideways, "
            "paper lamps glowing amber, cinematic illustration with rich texture"
        ),
    },
}

PROMPT_ORDER = [
    "archive-chapel",
    "gravity-orchard",
    "paper-nautilus",
    "ocean-typewriter",
    "clockmaker-desert",
    "monsoon-teahouse",
]


@dataclass
class GalleryJob:
    slug: str
    provider: str
    model: str
    title: str
    prompt: str
    size: str = "1024x1024"
    aspect_ratio: str = "1:1"
    provider_model_id: str | None = None


def build_provider_series(
    *,
    provider: str,
    model: str,
    provider_model_id: str,
    prefix: str,
) -> list[GalleryJob]:
    jobs: list[GalleryJob] = []
    for prompt_slug in PROMPT_ORDER:
        prompt = PROMPTS[prompt_slug]
        jobs.append(
            GalleryJob(
                slug=f"{prefix}-{prompt_slug}",
                provider=provider,
                model=model,
                provider_model_id=provider_model_id,
                title=prompt["title"],
                prompt=prompt["prompt"],
            )
        )
    return jobs


LIVE_GALLERY_JOBS: list[GalleryJob] = [
    GalleryJob(
        slug="fal-klein-archive-chapel",
        provider="Fal",
        model="FLUX Klein 4B",
        provider_model_id="fal-ai/flux-2/klein/4b/base",
        title=PROMPTS["archive-chapel"]["title"],
        prompt=PROMPTS["archive-chapel"]["prompt"],
    ),
    GalleryJob(
        slug="fal-schnell-gravity-orchard",
        provider="Fal",
        model="FLUX Schnell",
        provider_model_id="fal-ai/flux/schnell",
        title=PROMPTS["gravity-orchard"]["title"],
        prompt=PROMPTS["gravity-orchard"]["prompt"],
    ),
    GalleryJob(
        slug="fal-dev-paper-nautilus",
        provider="Fal",
        model="FLUX Dev",
        provider_model_id="fal-ai/flux/dev",
        title=PROMPTS["paper-nautilus"]["title"],
        prompt=PROMPTS["paper-nautilus"]["prompt"],
    ),
    GalleryJob(
        slug="fal-pro-monsoon-teahouse",
        provider="Fal",
        model="FLUX Pro 1.1",
        provider_model_id="fal-ai/flux-pro/v1.1",
        title=PROMPTS["monsoon-teahouse"]["title"],
        prompt=PROMPTS["monsoon-teahouse"]["prompt"],
    ),
    GalleryJob(
        slug="together-sd3-ocean-typewriter",
        provider="Together AI",
        model="Stable Diffusion 3 Medium",
        provider_model_id="stabilityai/stable-diffusion-3-medium",
        title=PROMPTS["ocean-typewriter"]["title"],
        prompt=PROMPTS["ocean-typewriter"]["prompt"],
    ),
    GalleryJob(
        slug="zai-glm-clockmaker-desert",
        provider="Z.AI",
        model="GLM Image",
        provider_model_id="glm-image",
        title=PROMPTS["clockmaker-desert"]["title"],
        prompt=PROMPTS["clockmaker-desert"]["prompt"],
    ),
]

RA1_GALLERY_JOBS = build_provider_series(
    provider="Netwrck",
    model="RA1",
    provider_model_id="ra1",
    prefix="ra1",
)

ZIMAGE_GALLERY_JOBS = build_provider_series(
    provider="Netwrck",
    model="ZImage",
    provider_model_id="zimage",
    prefix="zimage",
)

OPENAI_GALLERY_JOBS = build_provider_series(
    provider="OpenAI",
    model="GPT Image 2",
    provider_model_id="gpt-image-2",
    prefix="gpt-image-2",
)

JOB_SETS: dict[str, list[GalleryJob]] = {
    "live": LIVE_GALLERY_JOBS,
    "ra1": RA1_GALLERY_JOBS,
    "zimage": ZIMAGE_GALLERY_JOBS,
    "openai": OPENAI_GALLERY_JOBS,
}


def load_env_files() -> None:
    candidates = [
        ROOT / ".env",
        ROOT.parent / "netwrck" / ".env",
        Path.home() / ".secretbashrc",
    ]
    for path in candidates:
        if not path.exists():
            continue
        for raw_line in path.read_text().splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue
            if line.startswith("export "):
                line = line[len("export ") :]
            if "=" not in line:
                continue
            key, value = line.split("=", 1)
            key = key.strip()
            value = value.strip().strip("'").strip('"')
            if not key or value == "":
                continue
            os.environ.setdefault(key, value)


def slugify(value: str) -> str:
    value = value.lower().strip()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    return value.strip("-")


def ensure_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def fal_headers() -> dict[str, str]:
    key = os.environ.get("FAL_KEY") or os.environ.get("FAL_API_KEY")
    if not key:
        raise RuntimeError("Missing FAL_KEY / FAL_API_KEY")
    return {"Authorization": f"Key {key}", "Content-Type": "application/json"}


def generate_fal(job: GalleryJob) -> str:
    payload = {
        "prompt": job.prompt,
    }
    response = requests.post(
        f"https://fal.run/{job.provider_model_id}",
        headers=fal_headers(),
        json=payload,
        timeout=300,
    )
    response.raise_for_status()
    data = response.json()
    images = data.get("images") or []
    if not images or not images[0].get("url"):
        raise RuntimeError(f"Fal returned no image url for {job.slug}: {data}")
    return images[0]["url"]


def generate_together(job: GalleryJob) -> str:
    key = os.environ.get("TOGETHER_API_KEY")
    if not key:
        raise RuntimeError("Missing TOGETHER_API_KEY")
    payload = {
        "model": job.provider_model_id,
        "prompt": job.prompt,
        "size": job.size,
        "n": 1,
    }
    response = requests.post(
        "https://api.together.xyz/v1/images/generations",
        headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"},
        json=payload,
        timeout=300,
    )
    response.raise_for_status()
    data = response.json()
    result = data.get("data") or []
    if not result or not result[0].get("url"):
        raise RuntimeError(f"Together returned no image url for {job.slug}: {data}")
    return result[0]["url"]


def generate_zai(job: GalleryJob) -> str:
    key = os.environ.get("Z_API_KEY")
    if not key:
        raise RuntimeError("Missing Z_API_KEY")
    payload = {
        "model": job.provider_model_id,
        "prompt": job.prompt,
        "size": job.size,
    }
    response = requests.post(
        "https://api.z.ai/api/paas/v4/images/generations",
        headers={"Authorization": f"Bearer {key}"},
        json=payload,
        timeout=300,
    )
    response.raise_for_status()
    data = response.json()
    result = data.get("data") or []
    if not result or not result[0].get("url"):
        raise RuntimeError(f"Z.AI returned no image url for {job.slug}: {data}")
    return result[0]["url"]


def generate_ra1(job: GalleryJob) -> str:
    key = os.environ.get("NETWRCK_API_KEY")
    if not key:
        raise RuntimeError("Missing NETWRCK_API_KEY")
    payload = {
        "api_key": key,
        "prompt": job.prompt,
        "size": job.size,
    }
    response = requests.post(
        f"https://netwrck.com/api/{job.provider_model_id or 'ra1'}",
        json=payload,
        timeout=300,
    )
    if response.status_code != 200:
        return generate_fal_netwrck_fallback(job)
    data = response.json()
    image_url = data.get("image_url")
    if not image_url:
        return generate_fal_netwrck_fallback(job)
    return image_url


def generate_fal_netwrck_fallback(job: GalleryJob) -> str:
    endpoint = "fal-ai/flux/dev"
    payload: dict[str, Any] = {
        "prompt": f"Ultra realistic, {job.prompt}",
        "image_size": "square_hd",
        "num_images": 1,
        "num_inference_steps": 28,
        "guidance_scale": 3.5,
        "enable_safety_checker": False,
        "output_format": "png",
    }
    if job.provider_model_id == "zimage":
        endpoint = "fal-ai/z-image/turbo"
        payload = {
            "prompt": job.prompt,
            "image_size": "square_hd",
            "num_images": 1,
            "num_inference_steps": 8,
            "enable_safety_checker": False,
            "output_format": "webp",
            "acceleration": "regular",
        }

    response = requests.post(
        f"https://fal.run/{endpoint}",
        headers=fal_headers(),
        json=payload,
        timeout=300,
    )
    response.raise_for_status()
    data = response.json()
    if data.get("status") == "IN_QUEUE":
        response_url = data.get("response_url")
        if not response_url:
            raise RuntimeError(f"Fal fallback returned no response_url for {job.slug}: {data}")
        for _ in range(60):
            poll = requests.get(response_url, headers=fal_headers(), timeout=60)
            if poll.status_code == 202:
                import time

                time.sleep(2)
                continue
            poll.raise_for_status()
            data = poll.json()
            break
    images = data.get("images") or []
    if images and images[0].get("url"):
        return images[0]["url"]
    raise RuntimeError(f"Fal fallback returned no image url for {job.slug}: {data}")


def generate_zimage(job: GalleryJob) -> str:
    return generate_ra1(job)


def generate_ra1_legacy(job: GalleryJob) -> str:
    key = os.environ.get("NETWRCK_API_KEY")
    if not key:
        raise RuntimeError("Missing NETWRCK_API_KEY")
    payload = {
        "api_key": key,
        "prompt": job.prompt,
        "size": job.size,
    }
    response = requests.post(
        "https://netwrck.com/api/ra1-art-generator",
        json=payload,
        timeout=300,
    )
    response.raise_for_status()
    data = response.json()
    image_url = data.get("image_url")
    if not image_url:
        raise RuntimeError(f"RA1 returned no image url for {job.slug}: {data}")
    return image_url


def generate_openai(job: GalleryJob) -> str:
    key = os.environ.get("OPENAI_API_KEY")
    if not key:
        raise RuntimeError("Missing OPENAI_API_KEY")
    payload = {
        "model": job.provider_model_id,
        "prompt": job.prompt,
        "size": job.size,
    }
    response = requests.post(
        "https://api.openai.com/v1/images/generations",
        headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"},
        json=payload,
        timeout=300,
    )
    response.raise_for_status()
    data = response.json()
    items = data.get("data") or []
    if not items:
        raise RuntimeError(f"OpenAI returned no image payload for {job.slug}: {data}")
    if items[0].get("url"):
        return items[0]["url"]
    if items[0].get("b64_json"):
        return f"data:image/png;base64,{items[0]['b64_json']}"
    raise RuntimeError(f"OpenAI returned unsupported image payload for {job.slug}: {data}")


def generate_image(job: GalleryJob) -> str:
    if job.provider == "Fal":
        return generate_fal(job)
    if job.provider == "Together AI":
        return generate_together(job)
    if job.provider == "Z.AI":
        return generate_zai(job)
    if job.provider == "Netwrck":
        if job.provider_model_id == "zimage":
            return generate_zimage(job)
        return generate_ra1(job)
    if job.provider == "OpenAI":
        return generate_openai(job)
    raise RuntimeError(f"Unsupported provider: {job.provider}")


def download_image(url: str) -> bytes:
    if url.startswith("data:image/") and ";base64," in url:
        import base64

        return base64.b64decode(url.split(",", 1)[1])
    response = requests.get(url, timeout=300)
    response.raise_for_status()
    return response.content


def reencode_webp(raw: bytes) -> bytes:
    with Image.open(io.BytesIO(raw)) as img:
        converted = img.convert("RGB")
        out = io.BytesIO()
        converted.save(out, format="WEBP", quality=85, method=6)
        return out.getvalue()


def upload_webp(data: bytes, key: str) -> str:
    endpoint = os.environ.get("R2_ENDPOINT")
    bucket = os.environ.get("R2_BUCKET")
    access_key = os.environ.get("R2_ACCESS_KEY") or os.environ.get("CLOUDFLARE_R2_ACCESS_KEY_ID")
    secret_key = os.environ.get("R2_SECRET_KEY") or os.environ.get("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
    if not endpoint or not bucket or not access_key or not secret_key:
        raise RuntimeError("Missing R2 upload configuration")

    client = boto3.client(
        "s3",
        endpoint_url=endpoint,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        region_name="auto",
    )
    client.put_object(
        Bucket=bucket,
        Key=key,
        Body=data,
        ContentType="image/webp",
        CacheControl="public, max-age=31536000, immutable",
    )
    return f"{os.environ.get('R2_PUBLIC_URL') or DEFAULT_PUBLIC_BASE}/{key}"


def save_local_copy(data: bytes, path: Path) -> None:
    ensure_dir(path.parent)
    path.write_bytes(data)


def selected_jobs(job_ids: list[str], job_sets: list[str]) -> list[GalleryJob]:
    selected: list[GalleryJob] = []
    if not job_ids:
        if not job_sets:
            return LIVE_GALLERY_JOBS
        for job_set in job_sets:
            if job_set not in JOB_SETS:
                raise SystemExit(f"Unknown job set: {job_set}")
            selected.extend(JOB_SETS[job_set])
        return selected

    if not job_ids:
        return LIVE_GALLERY_JOBS
    lookup = {job.slug: job for jobs in JOB_SETS.values() for job in jobs}
    jobs = []
    for job_id in job_ids:
        if job_id not in lookup:
            raise SystemExit(f"Unknown job slug: {job_id}")
        jobs.append(lookup[job_id])
    return jobs


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Generate and upload art gallery images for the landing page."
    )
    parser.add_argument("--job", action="append", default=[], help="Specific job slug to run")
    parser.add_argument(
        "--set",
        action="append",
        default=[],
        help="Job set to run: live, ra1, zimage, openai",
    )
    parser.add_argument("--prefix", default=DEFAULT_PREFIX, help="Upload key prefix")
    parser.add_argument(
        "--output-dir",
        default=str(DEFAULT_OUTPUT_DIR),
        help="Directory for local WebP copies",
    )
    parser.add_argument(
        "--print-metadata",
        action="store_true",
        help="Print JSON metadata for the generated jobs",
    )
    return parser


def main() -> int:
    load_env_files()
    args = build_parser().parse_args()
    output_dir = Path(args.output_dir)
    ensure_dir(output_dir)

    metadata: list[dict[str, Any]] = []
    for job in selected_jobs(args.job, args.set):
        print(f"Generating {job.slug} [{job.provider} / {job.model}]...", file=sys.stderr)
        image_url = generate_image(job)
        raw = download_image(image_url)
        webp = reencode_webp(raw)
        object_key = f"{args.prefix}/{slugify(job.provider)}/{job.slug}.webp"
        public_url = upload_webp(webp, object_key)
        local_path = output_dir / f"{job.slug}.webp"
        save_local_copy(webp, local_path)
        item = {
            "slug": job.slug,
            "provider": job.provider,
            "model": job.model,
            "title": job.title,
            "prompt": job.prompt,
            "imageUrl": public_url,
            "localPath": str(local_path),
            "providerModelId": job.provider_model_id,
        }
        metadata.append(item)
        print(f"Uploaded {public_url}", file=sys.stderr)

    if args.print_metadata:
        print(json.dumps(metadata, indent=2))
    else:
        print(
            textwrap.dedent(
                """
                Completed gallery generation.
                Re-run with `--print-metadata` to dump the JSON payload for the UI.
                """
            ).strip()
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
