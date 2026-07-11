#!/usr/bin/env python3
"""Fake Codex TUI for auto-Skip doctests.

Signed protocol (PROTOCOL.md):
  1. Start on blocking update menu with › on 1. Update now
  2. CSI Down (\\x1b[B) re-renders with › on 2. Skip
  3. Enter (\\r) only after Skip is selected → residual banner + idle prompt
  4. Any line containing /status → parseable monthly/credits/reset fields

Without production auto-Skip (CSI Down + verify + Enter), FetchStatus hangs
on the menu (checkCodexWritable loading) until timeout — expected RED.
"""
from __future__ import annotations

import os
import sys

# Selection marker is U+203A (same as live Codex 0.143.0 snapshots).
MARK = "\u203a"

MENU_UPDATE_NOW = f"""  \u2728 \u200aUpdate available! 0.143.0 -> 0.144.0
  Release notes: https://github.com/openai/codex/releases/latest
{MARK} 1. Update now (runs `npm install -g @openai/codex`)
  2. Skip
  3. Skip until next version
  Press enter to continue
"""

MENU_SKIP = f"""  \u2728 \u200aUpdate available! 0.143.0 -> 0.144.0
  Release notes: https://github.com/openai/codex/releases/latest
  1. Update now (runs `npm install -g @openai/codex`)
{MARK} 2. Skip
  3. Skip until next version
  Press enter to continue
"""

# Residual banner still mentions update, but no menu options (non-blocking).
# No model:loading so waitForPrompt can become idle after menu dismiss.
IDLE_AFTER_SKIP = f"""\u256d\u2500\u2500 Update available! 0.143.0 -> 0.144.0 \u2500\u2500\u256e
\u2502 Run npm install -g @openai/codex to update.     \u2502
\u2570\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u256f
>_ OpenAI Codex (v0.143.0)
model:       gpt-5.5   /model to change
permissions: YOLO mode
{MARK} 
"""

# 42% left → ParseStatusSnapshot MonthlyUsage = 100-42 = 58% (same as other fakes).
STATUS_SCREEN = f"""Monthly credit limit: [######--------------] 42% left
                                      (resets 08:00 on 1 Aug)
                                      6,519 of 11,250 credits used
{MARK} 
"""


def write_screen(text: str) -> None:
    sys.stdout.write("\x1b[2J\x1b[H")
    sys.stdout.write(text)
    sys.stdout.flush()


def read_byte() -> bytes | None:
    ch = sys.stdin.buffer.read(1)
    if not ch:
        return None
    return ch


def main() -> int:
    # Optional marker dir for debugging inject sequences.
    marker_dir = os.environ.get("FAKE_CODEX_MARKER_DIR", "").strip()
    selection = "UPDATE_NOW"
    write_screen(MENU_UPDATE_NOW)

    line_buf = bytearray()
    while True:
        ch = read_byte()
        if ch is None:
            return 0

        # CSI Down: ESC [ B
        if ch == b"\x1b":
            rest = sys.stdin.buffer.read(2)
            if rest == b"[B":
                if marker_dir:
                    open(os.path.join(marker_dir, "csi-down"), "w").write("ok\n")
                selection = "SKIP"
                write_screen(MENU_SKIP)
                line_buf.clear()
                continue
            # Unknown ESC sequence — ignore.
            line_buf.clear()
            continue

        if ch in (b"\r", b"\n"):
            cmd = bytes(line_buf).decode("utf-8", errors="replace").strip()
            line_buf.clear()
            if selection == "SKIP" and (cmd == "" or cmd.lower() in ("",)):
                # Bare Enter after Skip dismisses the menu.
                if marker_dir:
                    open(os.path.join(marker_dir, "enter-skip"), "w").write("ok\n")
                write_screen(IDLE_AFTER_SKIP)
                selection = "IDLE"
                continue
            if selection == "UPDATE_NOW" and cmd == "":
                # Enter on Update now = silent upgrade path; hang so tests fail hard.
                if marker_dir:
                    open(os.path.join(marker_dir, "enter-update-now"), "w").write("BAD\n")
                sys.stdout.write("Updating… (fake refuses silent upgrade path)\n")
                sys.stdout.flush()
                while read_byte() is not None:
                    pass
                return 1
            if "/status" in cmd.lower() or cmd.lower().startswith("/status"):
                write_screen(STATUS_SCREEN)
                selection = "STATUS"
                continue
            # After idle, production sends "/status\n\r" which may arrive as "/status"
            if selection == "IDLE" and "status" in cmd.lower():
                write_screen(STATUS_SCREEN)
                selection = "STATUS"
                continue
            continue

        line_buf.extend(ch)
        # Production may inject "/status\n\r" as a single Send payload; also accept
        # when buffer already holds the command before CR.
        decoded = bytes(line_buf).decode("utf-8", errors="replace")
        if "/status" in decoded.lower() and selection in ("IDLE", "STATUS"):
            write_screen(STATUS_SCREEN)
            selection = "STATUS"
            line_buf.clear()


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except BrokenPipeError:
        raise SystemExit(0)
