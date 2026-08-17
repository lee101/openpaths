#!/usr/bin/env python3
"""Render the canonical OpenPaths road logo into browser and email assets."""

from pathlib import Path
import subprocess
import tempfile

from PIL import Image, ImageOps


ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / "openpaths-road-logo.svg"
PUBLIC = ROOT / "public"


def render_mark(size: int) -> Image.Image:
    with tempfile.NamedTemporaryFile(suffix=".png") as output:
        subprocess.run(
            [
                "convert",
                "-background",
                "none",
                str(SOURCE),
                "-resize",
                f"{size}x{size}",
                output.name,
            ],
            check=True,
        )
        return Image.open(output.name).convert("RGBA")


def white_mark_on_black(size: int) -> Image.Image:
    mark = render_mark(size)
    alpha = mark.getchannel("A")
    white = Image.new("RGBA", mark.size, "white")
    white.putalpha(alpha)
    tile = Image.new("RGBA", mark.size, "black")
    tile.alpha_composite(white)
    return tile


def save_webp(image: Image.Image, path: Path) -> None:
    image.save(path, "WEBP", lossless=True, method=6)


base = white_mark_on_black(512)
base.save(ROOT / "logo.png", "PNG", optimize=True)
base.resize((252, 252), Image.Resampling.LANCZOS).save(
    ROOT / "crawlers" / "openpaths-logo.png", "PNG", optimize=True
)

for size, name in ((256, "logo.webp"), (192, "logo-192.webp"), (512, "logo-512.webp")):
    save_webp(base.resize((size, size), Image.Resampling.LANCZOS), PUBLIC / name)

save_webp(base.resize((180, 180), Image.Resampling.LANCZOS), PUBLIC / "apple-touch-icon.webp")
save_webp(base.resize((16, 16), Image.Resampling.LANCZOS), PUBLIC / "favicon-16.webp")
save_webp(base.resize((32, 32), Image.Resampling.LANCZOS), PUBLIC / "favicon-32.webp")
base.resize((32, 32), Image.Resampling.LANCZOS).save(PUBLIC / "favicon.png", "PNG", optimize=True)
base.resize((48, 48), Image.Resampling.LANCZOS).save(PUBLIC / "favicon-48.png", "PNG", optimize=True)
base.save(PUBLIC / "favicon.ico", format="ICO", sizes=[(16, 16), (32, 32), (48, 48)])

print("Rendered OpenPaths logo assets from", SOURCE)
