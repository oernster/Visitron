"""Generate the icons from the master artwork in assets/, ported from ScreenState.

Three things come out of it.

The Windows icon: assets/application-icon.png becomes build/windows/icon.ico,
placed on the application, with the same file copied byte for byte to
installer/build/windows/icon.ico for the setup program, plus build/appicon.png, which
Wails reads. Both are squared by padding the trimmed artwork with transparency,
so Windows is never handed a frame that is not square.

The page icons: every other master becomes a picture under
frontend/src/assets/icons, trimmed of its transparent margin and written at
PAGE_WIDTH pixels wide with its own proportions, since squaring a drawing
somebody made would stretch it.

Nothing here resamples the artwork upwards. Reframing is a crop or a pad; an
upscale invents detail the master does not have.

Run it when a master changes:

    python tools/genicons.py

It is not part of the build. The output is committed, so a clone needs neither
Python nor Pillow to build anything.
"""

from __future__ import annotations

import pathlib
import shutil
import sys

try:
    from PIL import Image
except ImportError:  # pragma: no cover - a plain message beats a traceback
    sys.exit("Pillow is required: python -m pip install pillow")

# MASTER is the application's own artwork; the Windows icon comes from it.
MASTER = "application-icon.png"

# ICO_SIZES are the sizes Windows chooses between. Leaving one out makes
# Windows scale a neighbour, which looks soft.
ICO_SIZES = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]

# APPICON_SIZE is the square Wails reads from build/appicon.png.
APPICON_SIZE = 256

# PAGE_WIDTH is a page icon's width. The band draws its pictures far smaller;
# this leaves room for a high-density display without carrying a master's
# megabytes into the page.
PAGE_WIDTH = 256

REPO = pathlib.Path(__file__).resolve().parent.parent
MASTERS = REPO / "assets"
PAGE_ICONS = REPO / "frontend" / "src" / "assets" / "icons"
ICO = REPO / "build" / "windows" / "icon.ico"
# SETUP_ICO is the setup program's copy. Wails builds setup from its own
# folder, so it needs a file there; a copy of the bytes keeps the two the same.
SETUP_ICO = REPO / "installer" / "build" / "windows" / "icon.ico"
APPICON = REPO / "build" / "appicon.png"


def trimmed(master: pathlib.Path) -> Image.Image:
    """Open the master and crop away the transparent margin around its artwork."""
    image = Image.open(master).convert("RGBA")
    box = image.getbbox()
    return image.crop(box) if box is not None else image


def squared(artwork: Image.Image) -> Image.Image:
    """Pad the artwork with transparency to a square, centred."""
    side = max(artwork.size)
    square = Image.new("RGBA", (side, side))
    square.paste(artwork, ((side - artwork.width) // 2, (side - artwork.height) // 2))
    return square


def report(target: pathlib.Path) -> None:
    print(f"{target.relative_to(REPO)}  {target.stat().st_size:,} bytes")


def write_page_icon(artwork: Image.Image, target: pathlib.Path) -> None:
    """Write one page icon PAGE_WIDTH wide, keeping its proportions."""
    target.parent.mkdir(parents=True, exist_ok=True)
    height = max(1, round(artwork.height * PAGE_WIDTH / artwork.width))
    artwork.resize((PAGE_WIDTH, height), Image.LANCZOS).save(target, optimize=True)
    report(target)


def main() -> int:
    """Generate every icon, reporting what was written."""
    masters = sorted(MASTERS.glob("*.png"))
    if not (MASTERS / MASTER).exists():
        print(f"missing {(MASTERS / MASTER).relative_to(REPO)}", file=sys.stderr)
        return 1
    for master in masters:
        artwork = trimmed(master)
        if artwork.width < PAGE_WIDTH:
            print(f"{master.name} is narrower than {PAGE_WIDTH}; refusing to upscale it", file=sys.stderr)
            return 1
        write_page_icon(artwork, PAGE_ICONS / master.name)
        if master.name == MASTER:
            square = squared(artwork)
            square.save(ICO, format="ICO", sizes=ICO_SIZES)
            report(ICO)
            shutil.copyfile(ICO, SETUP_ICO)
            report(SETUP_ICO)
            square.resize((APPICON_SIZE, APPICON_SIZE), Image.LANCZOS).save(APPICON, optimize=True)
            report(APPICON)
    return 0


if __name__ == "__main__":
    sys.exit(main())
