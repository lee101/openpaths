#!/usr/bin/env python3
"""Generate consistent provider social cards with the OpenPaths road identity."""

from __future__ import annotations

from pathlib import Path
import subprocess
import tempfile
import textwrap

from PIL import Image, ImageDraw, ImageFont


W, H = 1200, 630
ROOT = Path(__file__).resolve().parent.parent
OUT_DIR = ROOT / "public" / "og"
LOGOS_DIR = ROOT / "public" / "logos"
BG = (5, 7, 11)
WHITE = (246, 248, 251)
OUT_DIR.mkdir(parents=True, exist_ok=True)

PROVIDERS = [
    ("anthropic", "Anthropic", "#d97706", "Claude Opus, Sonnet, and Haiku models"),
    ("openai", "OpenAI", "#10b981", "GPT, reasoning, realtime, image, and Codex models"),
    ("google", "Google", "#3b82f6", "Gemini multimodal models with long context"),
    ("xai", "xAI", "#8b5cf6", "Grok text, voice, and speech models"),
    ("deepseek", "DeepSeek", "#06b6d4", "Frontier reasoning and chat at efficient prices"),
    ("mistral", "Mistral", "#f97316", "General, coding, reasoning, and vision models"),
    ("minimax", "MiniMax", "#a78bfa", "Long-context chat plus image and video generation"),
    ("groq", "Groq", "#ef4444", "Low-latency LPU inference for open models"),
    ("together", "Together AI", "#22c55e", "Leading open models, image generation, and inference"),
    ("openrouter", "OpenRouter", "#64748b", "A broad catalog of hosted and free-tier models"),
    ("zai", "Z.AI", "#818cf8", "GLM chat, vision, reasoning, and image generation"),
    ("nous", "Nous Research", "#fb923c", "Hermes models with deep thinking and tool use"),
    ("fireworks", "Fireworks AI", "#fbbf24", "Fast serverless inference for leading open models"),
    ("fal", "Fal", "#34d399", "Serverless image, video, audio, and media models"),
    ("netwrck", "Netwrck", "#f472b6", "Image, anime, and video generation models"),
    ("textgenerator", "Text-Generator", "#38bdf8", "ModernBERT embeddings for RAG and semantic search"),
]


def rgb(hex_value: str) -> tuple[int, int, int]:
    value = hex_value.removeprefix("#")
    return tuple(int(value[index:index + 2], 16) for index in (0, 2, 4))


def mix(color: tuple[int, int, int], alpha: float) -> tuple[int, int, int]:
    return tuple(round(BG[index] * (1 - alpha) + color[index] * alpha) for index in range(3))


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


title_font = font("DejaVuSans-Bold.ttf", 72)
tag_font = font("DejaVuSans.ttf", 30)
mono = font("DejaVuSansMono.ttf", 17)
mono_bold = font("DejaVuSansMono-Bold.ttf", 16)


def make_image(slug: str, name: str, accent_hex: str, tagline: str) -> Path:
    accent = rgb(accent_hex)
    image = Image.new("RGB", (W, H), BG)
    draw = ImageDraw.Draw(image)

    # Restrained grid and provider-accent crop lines.
    for x in range(0, W, 60):
        draw.line((x, 0, x, H), fill=(9, 13, 19), width=1)
    for y in range(0, H, 60):
        draw.line((0, y, W, y), fill=(9, 13, 19), width=1)
    draw.rectangle((0, 0, W, 5), fill=accent)
    draw.rectangle((0, H - 4, W, H), fill=mix(accent, 0.28))

    # OpenPaths co-brand.
    openpaths = render_svg(LOGOS_DIR / "openpaths.svg", 38)
    if openpaths:
        image.paste(openpaths, (68, 59), openpaths)
    draw.text((116, 64), "OPENPATHS", font=mono_bold, fill=WHITE)
    draw.text((232, 64), "/  PROVIDER NETWORK", font=mono, fill=(96, 109, 126))

    # Left content column with guaranteed line lengths.
    draw.rounded_rectangle((68, 125, 292, 162), radius=18, fill=mix(accent, 0.14), outline=mix(accent, 0.48))
    draw.ellipse((88, 138, 98, 148), fill=accent)
    draw.text((110, 133), "AVAILABLE VIA API", font=mono_bold, fill=mix(accent, 0.92))
    draw.text((68, 201), name, font=title_font, fill=WHITE)

    y = 303
    for line in textwrap.wrap(tagline, width=39)[:2]:
        draw.text((70, y), line, font=tag_font, fill=(198, 207, 219))
        y += 43

    draw.text((70, 423), "One endpoint. Intelligent routing.", font=mono, fill=(118, 131, 148))
    draw.text((70, 453), "Usage-based billing across providers.", font=mono, fill=(118, 131, 148))

    # Provider identity tile, with the road mark used as a routing watermark.
    tile = (770, 79, 1127, 550)
    draw.rounded_rectangle(tile, radius=34, fill=(9, 13, 20), outline=mix(accent, 0.38), width=2)
    draw.ellipse((814, 139, 1084, 409), outline=mix(accent, 0.23), width=2)
    draw.ellipse((848, 173, 1050, 375), outline=mix(accent, 0.40), width=2)

    provider_logo = render_svg(LOGOS_DIR / f"{slug}.svg", 188)
    if provider_logo:
        image.paste(provider_logo, (949 - provider_logo.width // 2, 274 - provider_logo.height // 2), provider_logo)
    else:
        initials = {
            "minimax": "MM",
            "openrouter": "OR",
        }.get(slug, "".join(word[0] for word in name.split()[:2]).upper())
        fallback_font = font("DejaVuSans-Bold.ttf", 68)
        box = draw.textbbox((0, 0), initials, font=fallback_font)
        draw.text((949 - (box[2] - box[0]) / 2, 237), initials, font=fallback_font, fill=WHITE)

    # A small road mark acts as the co-brand signature inside the provider tile.
    if openpaths:
        signature = render_svg(LOGOS_DIR / "openpaths.svg", 62)
        image.paste(signature, (824, 444), signature)
    draw.text((897, 456), "ROUTED BY", font=mono, fill=(91, 104, 121))
    draw.text((897, 482), "OPENPATHS", font=mono_bold, fill=WHITE)

    draw.text((68, H - 40), f"openpaths.io/providers/{slug}", font=mono, fill=(75, 87, 102))
    output = OUT_DIR / f"og-{slug}.png"
    image.save(output, "PNG", optimize=True)
    return output


for provider in PROVIDERS:
    print(make_image(*provider))

print(f"Generated {len(PROVIDERS)} provider OG images in {OUT_DIR}")
