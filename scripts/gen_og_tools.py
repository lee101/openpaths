#!/usr/bin/env python3
"""Generate per-tool art and branded social cards for /tools.

Two assets per tool:
  public/og/tools/art/<slug>.webp  raw 3:2 generation used by the /tools grid
  public/og/tools/<slug>.webp      branded 1200x630 OG card built from that art

The art is rendered through our own RA1 image service (the same generation
stack omniserve-native fronts); the card is composited locally, so re-branding
after a logo change costs nothing but a --compose run.

Usage:
  python3 scripts/gen_og_tools.py              # render missing art, rebuild all cards
  python3 scripts/gen_og_tools.py --compose    # rebuild cards from existing art only
  python3 scripts/gen_og_tools.py --force      # re-render every piece of art
  python3 scripts/gen_og_tools.py --only lyria text-to-video
"""

from __future__ import annotations

import argparse
import io
import os
import subprocess
import sys
import tempfile
import textwrap
from pathlib import Path

import requests
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
OUT_DIR = ROOT / "public" / "og" / "tools"
ART_DIR = OUT_DIR / "art"
LOGO = ROOT / "public" / "logos" / "openpaths.svg"
ENDPOINT = "https://netwrck.com/api/ra1"
SIZE = "1152x768"
WEBP_QUALITY = 85

CARD_W, CARD_H = 1200, 630
BG = (5, 7, 11)
WHITE = (246, 248, 251)
MUTED = (150, 161, 178)
ACCENT = (56, 189, 248)

# RA1 prepends "Ultra realistic," which pulls hard toward photoreal portraits.
# The still-life framing plus the explicit exclusions keep every card on object
# subjects instead of a gallery of faces.
STYLE = (
    "still life object study with no living subject, dark editorial product photography, "
    "near-black background, soft teal and violet rim light, high detail, shallow depth of field, "
    "cinematic, centered composition, "
    "no people, no person, no human, no face, no portrait, no hands, no body, no character, "
    "no text, no words, no letters, no watermark"
)

# slug -> (art prompt subject, card title, card kicker)
TOOLS: dict[str, tuple[str, str, str]] = {
    "google-tts": (
        "a sculptural glowing audio waveform ribbon floating above a matte studio pedestal, two soft light sources suggesting a conversation",
        "Gemini Flash TTS",
        "SPEECH / 30 VOICES / MULTI-SPEAKER",
    ),
    "lyria": (
        "an abstract sculpture of flowing sheet music and glowing sound waves curling around a dark grand piano silhouette",
        "Lyria 3 Music Studio",
        "MUSIC / SONGS / LOOPS / OPUS",
    ),
    "text-to-image": (
        "a glowing paintbrush drawing a luminous cube of light out of dark empty space, particles trailing the stroke",
        "Text to Image",
        "IMAGE / AUTO IMAGE ENDPOINT",
    ),
    "image-edit": (
        "a framed photograph of a snowy mountain lake standing on dark glass, its right half repainted in vivid stylised color while the left half stays muted realism, split cleanly down the centre",
        "Image Style Transfer",
        "IMAGE EDIT / GPT IMAGE 2",
    ),
    "text-to-video": (
        "a clean 3d product render of a single long glowing filmstrip ribbon curving through completely empty black space, each small frame showing an uninhabited sand dune landscape, nothing else in frame",
        "Text to Video",
        "VIDEO / WAN 3.0 / NATIVE AUDIO",
    ),
    "image-to-video": (
        "a clean 3d product render of one framed photograph of an uninhabited mountain range floating in completely empty black space, the right edge of the picture stretching sideways into glowing motion-blur light trails, nothing else in frame",
        "Image to Video",
        "VIDEO / FIRST + LAST FRAME",
    ),
    "video-extension": (
        "a clean 3d product render of one narrow filmstrip in completely empty black space, its small frames showing an uninhabited coastline, the strip ending mid-frame and dissolving forward into a thin beam of light, dark and low key, nothing else in frame",
        "Video Extension",
        "VIDEO / GROK IMAGINE",
    ),
    "character-animator": (
        "a wooden artist mannequin mid-stride on a dark pedestal, glowing motion trails arcing off its limbs",
        "Character Animator",
        "VIDEO / WAN-ANIMATE",
    ),
    "music-generator": (
        "a glowing microphone surrounded by orbiting rings of waveform light in a dark recording studio",
        "Music Generator",
        "MUSIC / MINIMAX-MUSIC3",
    ),
    "remove-video-background": (
        "a glowing potted plant cleanly lifted off a grey checkerboard transparency backdrop, sharp cut edge, the backdrop peeling away behind it",
        "Background Remover",
        "VIDEO / TRANSPARENT WEBM",
    ),
    "image-to-3d": (
        "a flat photograph on the left rising into a faceted glowing 3D object on the right, wireframe seams visible",
        "Image to 3D",
        "3D / PIXAL3D / MESHY / TRIPO",
    ),
    "text-to-3d": (
        "glowing letterforms collapsing into a faceted 3D crystal object floating in dark space",
        "Text to 3D",
        "3D / AUTO IMAGE + PIXAL3D",
    ),
    "rig-3d": (
        "a wooden artist mannequin standing on a dark pedestal with a glowing skeleton of ball joints and thin struts overlaid along its wooden limbs",
        "3D Auto-Rigging",
        "3D / MESHY RIGGING / GLB + FBX",
    ),
    "retexture-3d": (
        "a smooth grey 3D vase half-covered by a rich patterned texture wrapping around its surface, split down the middle",
        "3D Retexture",
        "3D / TRELLIS-2",
    ),
    "playground": (
        "a clean 3d product render of nine floating frosted glass tiles arranged in a neat grid in completely empty black space, each tile glowing a different neon color, minimal geometric abstract, nothing else in frame",
        "Playground",
        "CHAT / IMAGE / VIDEO / AUDIO",
    ),
    "fusion": (
        "several glowing streams of light converging into one bright braided beam against a dark background",
        "Model Fusion",
        "PANEL / CONSENSUS / FUSE",
    ),
    "index": (
        "a dark workshop wall of glowing tool silhouettes arranged in a neat grid, each a different luminous color",
        "OpenPaths Tools",
        "IMAGE / VIDEO / MUSIC / SPEECH / 3D",
    ),
}

CARD_PATHS = {"index": "openpaths.io/tools"}

# Route per slug, for the card footer. Mirrors `path` in src/data/tools.ts.
TOOL_PATHS = {
    "google-tts": "/tools/google-tts",
    "lyria": "/tools/lyria",
    "text-to-image": "/text-to-image",
    "image-edit": "/image-edit",
    "text-to-video": "/text-to-video",
    "image-to-video": "/image-to-video",
    "video-extension": "/video-extension",
    "character-animator": "/character-animator",
    "music-generator": "/music-generator",
    "remove-video-background": "/remove-video-background",
    "image-to-3d": "/image-to-3d",
    "text-to-3d": "/text-to-3d",
    "rig-3d": "/rig-3d",
    "retexture-3d": "/retexture-3d",
    "playground": "/playground",
    "fusion": "/fusion",
}


def font(name: str, size: int) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(f"/usr/share/fonts/truetype/dejavu/{name}", size)


def render_svg(path: Path, size: int) -> Image.Image | None:
    if not path.exists():
        return None
    with tempfile.NamedTemporaryFile(suffix=".png") as output:
        result = subprocess.run(
            ["convert", "-background", "none", str(path), "-resize", f"{size}x{size}", output.name],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if result.returncode != 0:
            return None
        return Image.open(output.name).convert("RGBA")


def load_key() -> str:
    for candidate in (ROOT / ".env", ROOT.parent / "netwrck" / ".env", Path.home() / ".secretbashrc"):
        if not candidate.exists():
            continue
        for line in candidate.read_text().splitlines():
            line = line.strip().removeprefix("export ").strip()
            if line.startswith("NETWRCK_API_KEY="):
                return line.split("=", 1)[1].strip().strip("\"'")
    key = os.environ.get("NETWRCK_API_KEY", "")
    if not key:
        sys.exit("Missing NETWRCK_API_KEY")
    return key


def generate(key: str, slug: str, subject: str) -> bytes:
    response = requests.post(
        ENDPOINT,
        json={"api_key": key, "prompt": f"{subject}, {STYLE}", "size": SIZE},
        timeout=300,
    )
    response.raise_for_status()
    data = response.json()
    url = data.get("image_url")
    if not url:
        raise RuntimeError(f"{slug}: no image_url in {data}")
    image = requests.get(url, timeout=180)
    image.raise_for_status()
    return image.content


def cover(image: Image.Image, width: int, height: int) -> Image.Image:
    """Crop-to-fill, anchored on the centre of the frame."""
    scale = max(width / image.width, height / image.height)
    resized = image.resize((round(image.width * scale), round(image.height * scale)), Image.Resampling.LANCZOS)
    left = (resized.width - width) // 2
    top = (resized.height - height) // 2
    return resized.crop((left, top, left + width, top + height))


def compose_card(slug: str, title: str, kicker: str, art: Image.Image) -> Image.Image:
    """Full-bleed art at OG dimensions with a restrained brand lockup.

    The same file is the social card and the /tools grid thumbnail, so the
    lockup sits inside the bottom scrim where the grid already darkens the
    image and the card's own heading takes over.
    """
    card = cover(art, CARD_W, CARD_H).convert("RGB")

    # Bottom scrim: fully opaque under the lockup so the copy stays legible on
    # bright art, easing out over the art above it.
    band, solid = 320, 120
    scrim = Image.new("L", (1, CARD_H), 0)
    scrim_draw = ImageDraw.Draw(scrim)
    for offset in range(band):
        ramp = min(1.0, offset / (band - solid))
        scrim_draw.point((0, CARD_H - band + offset), fill=round(252 * ramp**1.5))
    card.paste(Image.new("RGB", (CARD_W, CARD_H), BG), (0, 0), scrim.resize((CARD_W, CARD_H)))

    draw = ImageDraw.Draw(card)
    draw.rectangle((0, 0, CARD_W, 5), fill=ACCENT)
    draw.rectangle((0, CARD_H - 4, CARD_W, CARD_H), fill=(16, 44, 58))

    mark = render_svg(LOGO, 34)
    if mark:
        card.paste(mark, (68, CARD_H - 148), mark)
    draw.text((112, CARD_H - 143), "OPENPATHS", font=font("DejaVuSansMono-Bold.ttf", 16), fill=WHITE)
    draw.text((228, CARD_H - 143), "/  " + kicker, font=font("DejaVuSansMono.ttf", 16), fill=ACCENT)

    url = CARD_PATHS.get(slug, f"openpaths.io{TOOL_PATHS.get(slug, '/tools')}")
    draw.text((CARD_W - 68, CARD_H - 129), url, font=font("DejaVuSansMono.ttf", 18), fill=MUTED, anchor="rs")

    title_font = font("DejaVuSans-Bold.ttf", 56)
    while draw.textlength(title, font=title_font) > CARD_W - 136 and title_font.size > 34:
        title_font = font("DejaVuSans-Bold.ttf", title_font.size - 4)
    draw.text((68, CARD_H - 46), title, font=title_font, fill=WHITE, anchor="ls")
    return card


def save_webp(image: Image.Image, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    image.save(path, "WEBP", quality=WEBP_QUALITY, method=6)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--force", action="store_true", help="re-render art that already exists")
    parser.add_argument("--compose", action="store_true", help="rebuild cards from existing art, no generation")
    parser.add_argument("--only", nargs="*", default=None, help="limit to these tool slugs")
    args = parser.parse_args()

    ART_DIR.mkdir(parents=True, exist_ok=True)
    slugs = args.only or list(TOOLS)
    key = None
    failures = []

    for slug in slugs:
        entry = TOOLS.get(slug)
        if entry is None:
            print(f"skip {slug}: unknown tool")
            continue
        subject, title, kicker = entry
        art_path = ART_DIR / f"{slug}.webp"

        if not art_path.exists() or (args.force and not args.compose):
            if args.compose:
                print(f"FAIL {slug}: no art at {art_path.relative_to(ROOT)}")
                failures.append(slug)
                continue
            key = key or load_key()
            try:
                raw = generate(key, slug, subject)
            except Exception as exc:  # noqa: BLE001 - report and continue the batch
                print(f"FAIL {slug}: {exc}")
                failures.append(slug)
                continue
            save_webp(Image.open(io.BytesIO(raw)).convert("RGB"), art_path)
            print(f"art   {art_path.relative_to(ROOT)} ({art_path.stat().st_size // 1024} KiB)")

        card_path = OUT_DIR / f"{slug}.webp"
        save_webp(compose_card(slug, title, kicker, Image.open(art_path).convert("RGB")), card_path)
        print(f"card  {card_path.relative_to(ROOT)} ({card_path.stat().st_size // 1024} KiB)")

    if failures:
        print(f"\n{len(failures)} failed: {', '.join(failures)}")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
