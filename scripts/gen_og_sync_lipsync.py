#!/usr/bin/env python3
"""Generate the sync-3 (sync-lipsync v3) model OG image -> public/og/og-sync-lipsync.png.

fal + OpenPaths co-branded: the real talking-avatar demo frame on the right inside a
rounded card with an audio-waveform motif, model name + tagline on the left.
"""
from PIL import Image, ImageDraw, ImageFont, ImageOps
import math, os, subprocess, tempfile

W, H = 1200, 630
HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, '..'))
OUT = os.path.join(ROOT, 'public', 'og', 'og-sync-lipsync.png')
LOGOS = os.path.join(ROOT, 'public', 'logos')
AVATAR = os.path.join(ROOT, 'tmp', 'sync-demo', 'frame.jpg')
ACCENT = (0x34, 0xd3, 0x99)  # emerald — fal accent / openpaths green
BG = (0x0a, 0x0a, 0x0a)


def font(path, size):
    try:
        return ImageFont.truetype(path, size)
    except OSError:
        return ImageFont.load_default()


F = '/usr/share/fonts/truetype/dejavu/'
f_huge = font(F + 'DejaVuSans-Bold.ttf', 76)
f_bold = font(F + 'DejaVuSans-Bold.ttf', 32)
f_tag = font(F + 'DejaVuSans.ttf', 24)
f_mono = font(F + 'DejaVuSansMono-Bold.ttf', 16)


def mix(rgb, alpha):
    a = alpha / 255.0
    return tuple(int(rgb[i] * a + BG[i] * (1 - a)) for i in range(3))


def render_svg(svg, size):
    with tempfile.NamedTemporaryFile(suffix='.png', delete=False) as tmp:
        out = tmp.name
    try:
        subprocess.run(['convert', '-background', 'none', svg, '-resize', f'{size}x{size}', out], check=True)
        return Image.open(out).convert('RGBA')
    finally:
        try:
            os.unlink(out)
        except OSError:
            pass


def paste_svg(img, svg, size, center):
    if not os.path.exists(svg):
        return
    logo = render_svg(svg, size)
    img.paste(logo, (center[0] - logo.width // 2, center[1] - logo.height // 2), logo)


def rounded(img, radius):
    mask = Image.new('L', img.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, img.width, img.height], radius=radius, fill=255)
    img.putalpha(mask)
    return img


img = Image.new('RGB', (W, H), BG)
d = ImageDraw.Draw(img)

# Keep the generator reproducible even when the original demo frame is not in the
# checkout: the committed OG contains the same 360px frame in this exact crop.
avatar_source = None
if os.path.exists(AVATAR):
    avatar_source = Image.open(AVATAR).convert('RGB')
elif os.path.exists(OUT):
    avatar_source = Image.open(OUT).convert('RGB').crop((790, 141, 1150, 501))

# quiet technical grid
for gx in range(0, W, 60):
    d.line((gx, 0, gx, H), fill='#0d1117', width=1)
for gy in range(0, H, 60):
    d.line((0, gy, W, gy), fill='#0d1117', width=1)

# accent frame bars
d.rectangle([0, 0, W, 4], fill=mix(ACCENT, 230))
d.rectangle([0, H - 4, W, H], fill=mix(ACCENT, 110))

# --- avatar card (right) ---
card_w, card_h = 360, 360
cx, cy = W - 230, H // 2 + 6
# glow rings behind card
for r, a in [(250, 22), (210, 34), (170, 54)]:
    d.ellipse([cx - r, cy - r, cx + r, cy + r], outline=mix(ACCENT, a), width=1)
if avatar_source is not None:
    av = ImageOps.fit(avatar_source, (card_w, card_h), centering=(0.5, 0.32))
    av = rounded(av, 28)
    img.paste(av, (cx - card_w // 2, cy - card_h // 2), av)
    d.rounded_rectangle([cx - card_w // 2, cy - card_h // 2, cx + card_w // 2, cy + card_h // 2],
                        radius=28, outline=mix(ACCENT, 150), width=2)

# audio waveform motif under the card
bars = 27
bw, gap = 5, 7
total = bars * (bw + gap) - gap
sx = cx - total // 2
by = cy + card_h // 2 + 26
for i in range(bars):
    h = 6 + int(26 * abs(math.sin(i * 0.6)) * (0.5 + 0.5 * math.cos(i * 0.3)))
    x = sx + i * (bw + gap)
    d.rounded_rectangle([x, by - h, x + bw, by + h], radius=2, fill=mix(ACCENT, 120 + (i % 5) * 22))

# --- left text column ---
x = 64
# OpenPaths brand line uses the canonical road mark.
paste_svg(img, os.path.join(LOGOS, 'openpaths.svg'), 38, (x + 19, 76))
d.text((x + 48, 68), 'OPENPATHS / MEDIA ROUTING', fill='#ffffff88', font=f_mono)

# co-brand pill: fal logo + via openpaths
paste_svg(img, os.path.join(LOGOS, 'fal.svg'), 30, (x + 16, 116))
d.text((x + 40, 108), 'fal', fill='#ffffffcc', font=f_bold)
d.text((x + 92, 113), 'via OpenPaths', fill=mix(ACCENT, 210), font=f_mono)

# title
d.text((x, 168), 'sync-3', fill='#ffffff', font=f_huge)
d.text((x, 256), 'Avatar Lip Sync', fill=ACCENT, font=f_huge)

# tagline
d.text((x, 360), 'One image + a voice track →', fill='#ffffffcc', font=f_bold)
d.text((x, 400), 'a talking, lip-synced character.', fill='#ffffffcc', font=f_bold)

# sub
d.text((x, 452), 'Image-to-video & video-to-video lip sync.', fill='#ffffff77', font=f_tag)

# bottom stats
d.text((x, H - 46), 'fal-ai/sync-lipsync/v3  |  $0.20 / sec  |  via openpaths.io',
       fill='#ffffff40', font=f_mono)

os.makedirs(os.path.dirname(OUT), exist_ok=True)
img.save(OUT, 'PNG', optimize=True)
print(OUT)
