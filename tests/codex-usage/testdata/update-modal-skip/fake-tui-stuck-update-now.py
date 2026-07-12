#!/usr/bin/env python3
"""Fake Codex TUI that never leaves 1. Update now.

Negative contract: production must NOT send Enter while selection is Update now.
CSI Down is ignored so verify-before-Enter fails; expected error is about not
being able to select Skip (or timeout), never status=ready / silent upgrade.
"""
from __future__ import annotations

import os
import sys

MARK = "\u203a"

MENU_UPDATE_NOW = f"""  \u2728 \u200aUpdate available! 0.143.0 -> 0.144.0
  Release notes: https://github.com/openai/codex/releases/latest
{MARK} 1. Update now (runs `npm install -g @openai/codex`)
  2. Skip
  3. Skip until next version
  Press enter to continue
"""


def write_screen(text: str) -> None:
    sys.stdout.write("\x1b[2J\x1b[H")
    sys.stdout.write(text)
    sys.stdout.flush()


def main() -> int:
    marker_dir = os.environ.get("FAKE_CODEX_MARKER_DIR", "").strip()
    write_screen(MENU_UPDATE_NOW)
    while True:
        ch = sys.stdin.buffer.read(1)
        if not ch:
            return 0
        if ch == b"\x1b":
            rest = sys.stdin.buffer.read(2)
            if rest == b"[B":
                # Acknowledge Down but refuse to move selection.
                if marker_dir:
                    open(os.path.join(marker_dir, "csi-down-ignored"), "a").write("1\n")
                write_screen(MENU_UPDATE_NOW)
            continue
        if ch in (b"\r", b"\n"):
            if marker_dir:
                open(os.path.join(marker_dir, "enter-while-update-now"), "w").write("BAD\n")
            # Stay on Update now forever (no upgrade, no status).
            write_screen(MENU_UPDATE_NOW)
            continue
        # Ignore other input (including /status typed into the modal).
        write_screen(MENU_UPDATE_NOW)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except BrokenPipeError:
        raise SystemExit(0)
