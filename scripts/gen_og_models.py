#!/usr/bin/env python3
"""Per-model 1200x630 social cards, cached by content hash (public/og/models)."""
from __future__ import annotations

import hashlib
import json
import subprocess
import sys
import tempfile
import textwrap
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

VERSION = "4"
W, H = 1200, 630
ROOT = Path(__file__).resolve().parent.parent
PUBLIC = ROOT / "public"
OUT = PUBLIC / "og" / "models"
MANIFEST = OUT / "manifest.json"
BG, WHITE, DIM, ACC = (5, 7, 11), (246, 248, 251), (150, 162, 178), (110, 231, 183)


def font(name: str, size: int):
    return ImageFont.truetype(f"/usr/share/fonts/truetype/dejavu/{name}", size)


F_TITLE = [font("DejaVuSans-Bold.ttf", s) for s in (76, 64, 54, 46)]
F_BODY = font("DejaVuSans.ttf", 30)
F_MONO = font("DejaVuSansMono.ttf", 22)
F_MONO_B = font("DejaVuSansMono-Bold.ttf", 20)
_logo_cache: dict[str, Image.Image | None] = {}


def load_logo(src: str, size: int) -> Image.Image | None:
    key = f"{src}@{size}"
    if key in _logo_cache:
        return _logo_cache[key]
    img = None
    path = PUBLIC / src.lstrip("/") if src.startswith("/") else None
    if path and path.exists():
        try:
            if path.suffix == ".svg":
                with tempfile.NamedTemporaryFile(suffix=".png") as o:
                    if subprocess.run(["rsvg-convert", "-a", "-w", str(size), "-h", str(size), "-o", o.name, str(path)],
                                      stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
                        img = Image.open(o.name).convert("RGBA")
            else:
                img = Image.open(path).convert("RGBA")
                img.thumbnail((size, size))
        except Exception:
            img = None
    _logo_cache[key] = img
    return img


def fit_title(draw, text: str, width: int):
    for f in F_TITLE:
        lines = textwrap.wrap(text, width=max(8, int(width / (f.size * 0.6))))
        if len(lines) <= 2 and all(draw.textlength(l, font=f) <= width for l in lines):
            return f, lines
    f = F_TITLE[-1]
    return f, textwrap.wrap(text, width=max(8, int(width / (f.size * 0.6))))[:3]


def render(m: dict, out: Path):
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)
    for x in range(0, W, 60):
        d.line((x, 0, x, H), fill=(10, 14, 20))
    for y in range(0, H, 60):
        d.line((0, y, W, y), fill=(10, 14, 20))
    d.rectangle((0, 0, W, 6), fill=ACC)

    op = load_logo("/logos/openpaths.svg", 36)
    if op:
        img.paste(op, (64, 52), op)
    d.text((112, 58), "OPENPATHS", font=F_MONO_B, fill=WHITE)
    d.text((238, 58), f"/  {m['provider'].upper()}"[:40], font=F_MONO, fill=DIM)

    f, lines = fit_title(d, m["name"], 720)
    y = 150
    for line in lines:
        d.text((64, y), line, font=f, fill=WHITE)
        y += int(f.size * 1.12)
    y += 18
    d.text((64, y), m["price"][:48], font=F_BODY, fill=(210, 218, 228))
    y += 48
    if m.get("context") and m["context"].upper() not in ("N/A", "NA", "-"):
        d.text((64, y), f"Context {m['context']}"[:48], font=F_BODY, fill=DIM)
        y += 48
    tags = "  ".join(f"#{t.replace(' ', '-')}" for t in m.get("tags", []))
    if tags:
        d.text((64, y + 6), tags[:56], font=F_MONO, fill=ACC)

    box = (64, H - 118, 64 + min(760, 40 + int(d.textlength('model="' + m["id"] + '"', font=F_MONO))), H - 64)
    d.rounded_rectangle(box, radius=10, fill=(14, 19, 27), outline=(40, 50, 64))
    d.text((84, H - 104), f'model="{m["id"]}"'[:58], font=F_MONO, fill=WHITE)

    tile = (850, 150, 1136, 436)
    d.rounded_rectangle(tile, radius=32, fill=(11, 15, 22), outline=(46, 58, 74), width=2)
    logo = load_logo(m.get("logo", ""), 170)
    if logo:
        img.paste(logo, (993 - logo.width // 2, 293 - logo.height // 2), logo)
    else:
        ini = "".join(w[0] for w in m["provider"].split()[:2]).upper() or "AI"
        fb = F_TITLE[0]
        bb = d.textbbox((0, 0), ini, font=fb)
        d.text((993 - (bb[2] - bb[0]) / 2, 250), ini, font=fb, fill=WHITE)
    d.text((850, 470), "One API. Any model.", font=F_MONO, fill=DIM)
    d.text((850, 504), "openpaths.io", font=F_MONO_B, fill=WHITE)
    img.quantize(colors=128, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE).save(out, "PNG", optimize=True)


def main():
    rows = json.loads(Path(sys.argv[1]).read_text())
    OUT.mkdir(parents=True, exist_ok=True)
    manifest = json.loads(MANIFEST.read_text()) if MANIFEST.exists() else {}
    made = 0
    for m in rows:
        h = hashlib.sha1((VERSION + json.dumps(m, sort_keys=True)).encode()).hexdigest()[:16]
        out = OUT / f"{m['slug']}.png"
        if manifest.get(m["slug"]) == h and out.exists():
            continue
        render(m, out)
        manifest[m["slug"]] = h
        made += 1
    MANIFEST.write_text(json.dumps(manifest, indent=0, sort_keys=True))
    print(f"model OG: {made} generated, {len(rows) - made} cached")


if __name__ == "__main__":
    main()
