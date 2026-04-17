#!/usr/bin/env python3
"""Generate the homepage OG image for openpaths.io.

Restores the 'Search and you shall find open pathways' theme with the
OpenPaths logo centered visually. Writes to public/og-image.png.
"""
from PIL import Image, ImageDraw, ImageFont
import math, os, subprocess, tempfile

W, H = 1200, 630

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.abspath(os.path.join(HERE, "..", "public", "og-image.png"))
LOGO_SVG = os.path.abspath(os.path.join(HERE, "..", "public", "logos", "openpaths.svg"))


def render_svg(svg_path, size):
    """Render an SVG to PNG bytes at the given size via rsvg-convert."""
    with tempfile.NamedTemporaryFile(suffix=".png", delete=False) as tmp:
        out = tmp.name
    try:
        subprocess.run(
            ["rsvg-convert", "-w", str(size), "-h", str(size), "-o", out, svg_path],
            check=True,
        )
        return Image.open(out).convert("RGBA")
    finally:
        try:
            os.unlink(out)
        except OSError:
            pass


img = Image.new("RGB", (W, H), "#000000")
draw = ImageDraw.Draw(img)

# --- decorative concentric rings (right-aligned) ---
cx, cy = W - 220, H // 2
for r, alpha in [(360, 0x10), (280, 0x14), (200, 0x1a), (120, 0x22)]:
    draw.ellipse(
        [cx - r, cy - r, cx + r, cy + r],
        outline=f"#ffffff{alpha:02x}", width=1,
    )

# radial dots on one ring, accent
for angle_deg in range(0, 360, 8):
    rad = math.radians(angle_deg)
    px = int(cx + 200 * math.cos(rad))
    py = int(cy + 200 * math.sin(rad))
    draw.ellipse([px - 1, py - 1, px + 2, py + 2], fill="#22c55e60")

# subtle grid dots
for i in range(0, W, 40):
    for j in range(0, H, 40):
        dist = math.sqrt((i - cx) ** 2 + (j - cy) ** 2)
        if 195 < dist < 205 or 275 < dist < 285 or 355 < dist < 365:
            draw.rectangle([i, j, i + 1, j + 1], fill="#ffffff18")

# --- fonts ---
def _font(path, size, fallback=None):
    try:
        return ImageFont.truetype(path, size)
    except OSError:
        return fallback or ImageFont.load_default()

font_bold  = _font("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 66)
font_bold2 = _font("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 66)
font_light = _font("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 24)
font_mono  = _font("/usr/share/fonts/truetype/firacode/FiraCode-Bold.ttf", 16)
font_sub   = _font("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 20)

x = 72

# brand line with green dot
draw.ellipse([x, 84, x + 10, 94], fill="#22c55e")
draw.text((x + 18, 82), "openpaths.io", fill="#ffffff66", font=font_mono)

# OpenPaths logo in the right-hand circles
if os.path.exists(LOGO_SVG):
    logo = render_svg(LOGO_SVG, 180)
    img.paste(logo, (cx - 90, cy - 90), logo)

# main heading
y = 170
draw.text((x, y), "The Open Source", fill="#ffffff", font=font_bold)
y += 82
draw.text((x, y), "Model Router.", fill="#ffffff66", font=font_bold2)

# description
y += 96
draw.text(
    (x, y),
    "Search and you shall find open pathways.",
    fill="#22c55eee",
    font=font_light,
)

y += 42
draw.text(
    (x, y),
    "Newly learned pathways for millisecond routing",
    fill="#ffffffaa",
    font=font_sub,
)
y += 28
draw.text(
    (x, y),
    "between large model providers & art generators.",
    fill="#ffffffaa",
    font=font_sub,
)

# bottom bar + stats
draw.rectangle([0, H - 3, W, H], fill="#ffffff0d")
draw.rectangle([0, 0, W, 2], fill="#ffffff0d")
draw.text(
    (x, H - 46),
    "500+ Models  |  15+ Providers  |  <50ms Router Latency",
    fill="#ffffff40",
    font=font_mono,
)

img.save(OUT, "PNG", optimize=True)
print(f"saved {OUT} ({img.size})")
