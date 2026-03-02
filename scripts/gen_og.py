#!/usr/bin/env python3
from PIL import Image, ImageDraw, ImageFont
import math

W, H = 1200, 630
img = Image.new("RGB", (W, H), "#000000")
draw = ImageDraw.Draw(img)

# concentric circles (decorative, like the site)
cx, cy = W // 2 + 200, H // 2
for r in [320, 240, 160]:
    draw.ellipse(
        [cx - r, cy - r, cx + r, cy + r],
        outline="#ffffff0d", width=1
    )

# subtle radial dots / grid accent
for i in range(0, W, 40):
    for j in range(0, H, 40):
        dist = math.sqrt((i - cx) ** 2 + (j - cy) ** 2)
        if 155 < dist < 165 or 235 < dist < 245:
            a = max(0, min(20, int(25 - dist / 30)))
            if a > 0:
                draw.rectangle([i, j, i + 1, j + 1], fill=f"#ffffff{a:02x}")

# fonts
try:
    font_bold = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 62)
    font_light = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 22)
    font_mono = ImageFont.truetype("/usr/share/fonts/truetype/firacode/FiraCode-Bold.ttf", 16)
    font_sub = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 19)
except:
    font_bold = ImageFont.load_default()
    font_light = ImageFont.load_default()
    font_mono = ImageFont.load_default()
    font_sub = ImageFont.load_default()

x_start = 72
y = 80

# green dot + version badge
draw.ellipse([x_start, y + 4, x_start + 8, y + 12], fill="#22c55e")
draw.text((x_start + 16, y), "openpaths.io", fill="#ffffff66", font=font_mono)

y = 160

# main heading
draw.text((x_start, y), "The Open Source", fill="#ffffff", font=font_bold)
y += 76
draw.text((x_start, y), "Model Router.", fill="#ffffff66", font=font_bold)

y += 100

# description
draw.text(
    (x_start, y),
    "Search and we shall find open pathways.",
    fill="#ffffffcc",
    font=font_light,
)
y += 36
draw.text(
    (x_start, y),
    "Newly learned pathways for millisecond routing",
    fill="#ffffff99",
    font=font_sub,
)
y += 30
draw.text(
    (x_start, y),
    "between large model providers or art generators.",
    fill="#ffffff99",
    font=font_sub,
)

# bottom bar accent line
draw.rectangle([0, H - 3, W, H], fill="#ffffff0d")
# bottom left branding
draw.text((x_start, H - 44), "432+ Models  |  15+ Providers  |  <50ms Latency", fill="#ffffff40", font=font_mono)

# thin accent line at top
draw.rectangle([0, 0, W, 2], fill="#ffffff0d")

out = "/vfast/data/code/openpath/public/og-image.png"
img.save(out, "PNG", optimize=True)
print(f"saved {out} ({img.size})")
