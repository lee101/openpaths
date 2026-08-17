#!/usr/bin/env python3
"""Generate the OpenPaths homepage social card."""

from pathlib import Path
import subprocess
import tempfile

from PIL import Image, ImageDraw, ImageFont


W, H = 1200, 630
ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "public" / "og-image.png"
LOGO = ROOT / "public" / "logos" / "openpaths.svg"
BG = (5, 7, 11)
PANEL = (10, 14, 21)
WHITE = (246, 248, 251)
MUTED = (147, 158, 175)
GREEN = (52, 211, 153)


def font(name: str, size: int) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(f"/usr/share/fonts/truetype/dejavu/{name}", size)


def render_svg(path: Path, size: int) -> Image.Image:
    with tempfile.NamedTemporaryFile(suffix=".png") as output:
        subprocess.run(
            ["convert", "-background", "none", str(path), "-resize", f"{size}x{size}", output.name],
            check=True,
        )
        return Image.open(output.name).convert("RGBA")


headline = font("DejaVuSans-Bold.ttf", 72)
body = font("DejaVuSans.ttf", 25)
body_bold = font("DejaVuSans-Bold.ttf", 24)
mono = font("DejaVuSansMono.ttf", 17)
mono_bold = font("DejaVuSansMono-Bold.ttf", 16)

image = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(image)

# Quiet technical grid, clipped visually by the right identity panel.
for x in range(0, W, 48):
    draw.line((x, 0, x, H), fill=(11, 16, 23), width=1)
for y in range(0, H, 48):
    draw.line((0, y, W, y), fill=(11, 16, 23), width=1)

# Brand line.
small_logo = render_svg(LOGO, 38)
image.paste(small_logo, (68, 61), small_logo)
draw.text((116, 66), "OPENPATHS", font=mono_bold, fill=WHITE)
draw.text((232, 66), "/  OPEN MODEL ROUTING", font=mono, fill=(101, 113, 130))

# Core proposition.
draw.text((68, 160), "One API.", font=headline, fill=WHITE)
draw.text((68, 242), "Every path.", font=headline, fill=GREEN)
draw.text((72, 351), "Route each request to the right model,", font=body, fill=(203, 211, 222))
draw.text((72, 389), "provider, or generation tool.", font=body, fill=(203, 211, 222))

# Product qualities use real dividers instead of a stale provider/model count.
labels = ("OPEN SOURCE", "OPENAI-COMPATIBLE", "PAY AS YOU GO")
x = 72
for index, label in enumerate(labels):
    if index:
        draw.ellipse((x, 487, x + 5, 492), fill=(70, 82, 98))
        x += 23
    draw.text((x, 480), label, font=mono_bold, fill=(137, 149, 165))
    x += draw.textlength(label, font=mono_bold) + 23

# Right-hand identity panel.
panel = (760, 62, 1132, 568)
draw.rounded_rectangle(panel, radius=34, fill=PANEL, outline=(31, 40, 52), width=2)
draw.rounded_rectangle((783, 86, 1109, 544), radius=28, outline=(20, 55, 47), width=1)
draw.text((812, 112), "SMART ROUTING", font=mono_bold, fill=GREEN)
draw.text((812, 139), "REQUEST → BEST AVAILABLE PATH", font=mono, fill=(88, 102, 119))

# A green entry route leads into the canonical mark.
draw.line((790, 324, 857, 324), fill=(20, 71, 58), width=8)
draw.ellipse((783, 317, 797, 331), fill=GREEN)
mark = render_svg(LOGO, 294)
image.paste(mark, (820, 181), mark)

draw.rounded_rectangle((812, 485, 1080, 520), radius=17, fill=(9, 35, 30), outline=(24, 88, 71))
draw.ellipse((832, 497, 842, 507), fill=GREEN)
draw.text((852, 493), "PATH FOUND", font=mono_bold, fill=(186, 246, 221))

# Edge accents give the card a clean crop in social feeds.
draw.rectangle((0, 0, W, 4), fill=GREEN)
draw.rectangle((0, H - 4, W, H), fill=(14, 44, 37))
draw.text((68, H - 39), "openpaths.io", font=mono, fill=(84, 97, 113))

image.save(OUT, "PNG", optimize=True)
print(f"saved {OUT} ({image.size[0]}x{image.size[1]})")
