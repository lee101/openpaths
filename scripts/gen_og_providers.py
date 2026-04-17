#!/usr/bin/env python3
"""Generate per-provider OG images for openpaths.io/og/*.png.

Each image pairs the provider's real logo (rendered from public/logos/<slug>.svg)
with OpenPaths branding, so social shares on provider pages look like
"this is Provider X, available via OpenPaths".
"""
from PIL import Image, ImageDraw, ImageFont
import math, os, subprocess, tempfile

W, H = 1200, 630
HERE = os.path.dirname(os.path.abspath(__file__))
OUT_DIR = os.path.abspath(os.path.join(HERE, '..', 'public', 'og'))
LOGOS_DIR = os.path.abspath(os.path.join(HERE, '..', 'public', 'logos'))
os.makedirs(OUT_DIR, exist_ok=True)

PROVIDERS = [
    # slug, display_name, accent_hex, tagline, logo_slug (optional)
    ('anthropic',    'Anthropic',       '#d97706', 'Claude Opus 4.7, Sonnet 4.6 & Haiku 4.5'),
    ('openai',       'OpenAI',          '#10b981', 'GPT-5 Chat, o3/o4-mini & Codex'),
    ('google',       'Google',          '#3b82f6', 'Gemini 3.1 Pro & 2.5 Flash · 2M context'),
    ('xai',          'xAI',             '#8b5cf6', 'Grok 4, Grok 4.1 Fast · 2M context'),
    ('deepseek',     'DeepSeek',        '#06b6d4', 'V3.2 & Reasoner · frontier at low cost'),
    ('mistral',      'Mistral',         '#f97316', 'Large 3, Codestral, Magistral & Devstral'),
    ('minimax',      'MiniMax',         '#a78bfa', 'M2.7, M2.5 · 1M context · Hailuo video'),
    ('groq',         'Groq',            '#ef4444', 'LPU-accelerated Llama & Mixtral inference'),
    ('together',     'Together AI',     '#22c55e', 'Qwen, Kimi, GLM-5.1, FLUX & open models'),
    ('openrouter',   'OpenRouter',      '#64748b', '600+ models including free-tier providers'),
    ('zai',          'Z.AI',            '#818cf8', 'GLM-5.1, GLM-4.7 vision & GLM Image gen'),
    ('nous',         'Nous Research',   '#fb923c', 'Hermes 4 70B & 405B · deep thinking'),
    ('fireworks',    'Fireworks AI',    '#fbbf24', 'Fast inference · GPT-OSS 120B & GLM-5'),
    ('fal',          'Fal',             '#34d399', 'Serverless image & video · FLUX, Kling, Luma'),
    ('netwrck',      'Netwrck',         '#f472b6', 'RA1, ZImage anime & Wan/LTX/RA2V video'),
    ('textgenerator','Text-Generator',  '#38bdf8', 'ModernBERT embeddings for RAG & search'),
]


def hex_to_rgb(h):
    h = h.lstrip('#')
    return tuple(int(h[i:i+2], 16) for i in (0, 2, 4))


def load_font(path, size):
    try:
        return ImageFont.truetype(path, size)
    except OSError:
        return ImageFont.load_default()


font_huge = load_font("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 84)
font_bold = load_font("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 34)
font_tag  = load_font("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 24)
font_mono = load_font("/usr/share/fonts/truetype/firacode/FiraCode-Bold.ttf", 16)
font_small= load_font("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 18)


def render_svg(svg_path, size):
    """Render an SVG to a PIL RGBA Image at the given pixel dimensions."""
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


def paste_svg(img, svg_path, box_size, center_xy):
    if not os.path.exists(svg_path):
        return False
    logo = render_svg(svg_path, box_size)
    cx, cy = center_xy
    img.paste(logo, (cx - logo.width // 2, cy - logo.height // 2), logo)
    return True


def mix(rgb, bg, alpha):
    """Alpha-blend an RGB color over a background at 0..255 alpha."""
    a = alpha / 255.0
    return tuple(int(rgb[i] * a + bg[i] * (1 - a)) for i in range(3))


def make_image(slug, name, accent, tagline):
    bg = (0x0a, 0x0a, 0x0a)
    img = Image.new("RGB", (W, H), bg)
    draw = ImageDraw.Draw(img)
    r, g, b = hex_to_rgb(accent)
    accent_rgb = (r, g, b)

    # --- decorative concentric circles (right) ---
    cx, cy = W - 200, H // 2
    for radius, alpha in [(340, 24), (260, 40), (180, 70), (100, 110)]:
        draw.ellipse(
            [cx - radius, cy - radius, cx + radius, cy + radius],
            outline=mix(accent_rgb, bg, alpha), width=1,
        )

    # accent dotted ring
    dot_col = mix(accent_rgb, bg, 180)
    for angle_deg in range(0, 360, 7):
        rad = math.radians(angle_deg)
        px = int(cx + 180 * math.cos(rad))
        py = int(cy + 180 * math.sin(rad))
        draw.ellipse([px - 2, py - 2, px + 3, py + 3], fill=dot_col)

    # top & bottom accent bars
    draw.rectangle([0, 0, W, 4], fill=mix(accent_rgb, bg, 230))
    draw.rectangle([0, H - 4, W, H], fill=mix(accent_rgb, bg, 110))

    # --- provider logo, centered in the rings ---
    provider_logo = os.path.join(LOGOS_DIR, f"{slug}.svg")
    paste_svg(img, provider_logo, 200, (cx, cy))

    # --- small OpenPaths co-brand in top-right of image ---
    op_logo = os.path.join(LOGOS_DIR, "openpaths.svg")
    paste_svg(img, op_logo, 44, (W - 74, 64))
    draw.text((W - 230, 56), "via openpaths", fill="#ffffff88", font=font_mono)

    x = 72

    # openpaths brand line
    draw.ellipse([x, 82, x + 10, 92], fill="#22c55e")
    draw.text((x + 18, 80), "openpaths.io", fill="#ffffff55", font=font_mono)

    # "AVAILABLE VIA OPENPATHS" pill — subtle fill, visible outline, white label
    pill_fill = mix(accent_rgb, bg, 40)
    pill_stroke = mix(accent_rgb, bg, 170)
    pill_x, pill_y = x, 114
    pw, ph = 220, 30
    draw.rounded_rectangle(
        [pill_x, pill_y, pill_x + pw, pill_y + ph],
        radius=6, fill=pill_fill, outline=pill_stroke, width=1,
    )
    draw.text((pill_x + 14, pill_y + 6), "AVAILABLE VIA OPENPATHS", fill="#ffffff", font=font_mono)

    # provider name — large
    y = 186
    draw.text((x, y), name, fill="#ffffff", font=font_huge)
    y += 100

    # tagline (accent color, fully opaque)
    draw.text((x, y), tagline, fill=accent_rgb, font=font_bold)
    y += 54

    # sub description
    draw.text(
        (x, y),
        "Route intelligently. Pay only for what you use.",
        fill="#ffffff88",
        font=font_tag,
    )

    # bottom stats
    draw.text(
        (x, H - 44),
        "500+ Models  |  15+ Providers  |  <50ms Latency",
        fill="#ffffff35",
        font=font_mono,
    )

    out = os.path.join(OUT_DIR, f"og-{slug}.png")
    img.save(out, "PNG", optimize=True)
    return out


for slug, name, accent, tagline in PROVIDERS:
    path = make_image(slug, name, accent, tagline)
    print(f"  {path}")

print(f"\nGenerated {len(PROVIDERS)} provider OG images in {OUT_DIR}/")
