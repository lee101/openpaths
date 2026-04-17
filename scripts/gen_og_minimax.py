#!/usr/bin/env python3
"""Generate OG image featuring MiniMax M2.7 on OpenPaths."""
from PIL import Image, ImageDraw, ImageFont
import math, sys

W, H = 1200, 630
img = Image.new("RGB", (W, H), "#0a0a0a")
draw = ImageDraw.Draw(img)

# --- decorative concentric circles (right side) ---
cx, cy = W - 180, H // 2
for r, alpha in [(340, "08"), (260, "0e"), (180, "16"), (100, "20")]:
    draw.ellipse([cx - r, cy - r, cx + r, cy + r], outline=f"#ffffff{alpha}", width=1)

# subtle arc highlight (top-right quadrant of innermost circle)
for angle_deg in range(-30, 80):
    rad = math.radians(angle_deg)
    r = 100
    px = int(cx + r * math.cos(rad))
    py = int(cy + r * math.sin(rad))
    draw.rectangle([px, py, px + 2, py + 2], fill="#a78bfa40")

# radial dots on the 180 ring
for angle_deg in range(0, 360, 8):
    rad = math.radians(angle_deg)
    px = int(cx + 180 * math.cos(rad))
    py = int(cy + 180 * math.sin(rad))
    draw.rectangle([px, py, px + 2, py + 2], fill="#ffffff18")

# --- fonts ---
try:
    font_huge  = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 74)
    font_bold  = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 48)
    font_med   = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 24)
    font_small = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 20)
    font_mono  = ImageFont.truetype("/usr/share/fonts/truetype/firacode/FiraCode-Bold.ttf", 16)
except Exception:
    font_huge = font_bold = font_med = font_small = font_mono = ImageFont.load_default()

x = 72

# --- top branding bar ---
draw.ellipse([x, 80, x + 10, 90], fill="#22c55e")
draw.text((x + 18, 78), "openpaths.io", fill="#ffffff55", font=font_mono)

# --- "NEW MODEL" badge ---
badge_x, badge_y = x, 128
bw, bh = 160, 32
draw.rounded_rectangle([badge_x, badge_y, badge_x + bw, badge_y + bh], radius=6, fill="#7c3aed22", outline="#7c3aed88", width=1)
draw.text((badge_x + 14, badge_y + 6), "NEW  MODEL", fill="#a78bfa", font=font_mono)

# --- main headline ---
y = 188
draw.text((x, y), "MiniMax M2.7", fill="#ffffff", font=font_huge)
y += 86

# --- sub line ---
draw.text((x, y), "Now available on OpenPaths", fill="#a78bfacc", font=font_bold)
y += 66

# --- feature pills row ---
PILLS = [("1M Context", "#22c55e"), ("Tool Use", "#3b82f6"), ("Streaming", "#f59e0b")]
px_pill = x
for label, color in PILLS:
    pw = len(label) * 11 + 24
    draw.rounded_rectangle([px_pill, y, px_pill + pw, y + 28], radius=5, fill=f"{color}22", outline=f"{color}88", width=1)
    draw.text((px_pill + 12, y + 5), label, fill=f"{color}ee", font=font_mono)
    px_pill += pw + 12

y += 54

# --- description ---
draw.text((x, y), "Route to the world's best models in milliseconds.", fill="#ffffffaa", font=font_med)

# --- bottom bar ---
draw.rectangle([0, H - 3, W, H], fill="#7c3aed33")
draw.rectangle([0, 0, W, 2], fill="#ffffff0d")
draw.text((x, H - 44), "500+ Models  |  15+ Providers  |  <50ms Latency", fill="#ffffff35", font=font_mono)

# --- save ---
out = sys.argv[1] if len(sys.argv) > 1 else "/nvme0n1-disk/code/openpaths/public/og-image.png"
img.save(out, "PNG", optimize=True)
print(f"saved {out} ({img.size})")
