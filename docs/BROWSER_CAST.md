# Browser Cast

`/browser` puts a real Chrome window on the phone. Unlike `/screen`, which
mirrors the whole desktop and needs the Mac to be awake and unlocked, `/browser`
drives one Chrome window through the
[browser-agent](https://github.com/xhd2015/browser-agent) extension, so pages
render on the host and you can still log in, scan a QR code, or click through a
consent screen while the Mac is locked.

- `GET /browser` — the page (React, embedded in the binary)
- `GET /api/browser/status` — daemon, session, viewport and capture state
- `POST /api/browser/ensure` — bring the daemon, session, window and tab up
- `POST /api/browser/navigate` — `{"url": "..."}` in the driven tab
- `POST /api/browser/preset` — `{"preset": "iphone"|"ipad"|"mac"}`
- `POST /api/browser/mode` — `{"mode": "cast"|"shot"}`
- `POST /api/browser/refresh` — force one capture now (shot mode)
- `GET /api/browser/sessions` — live sessions and their tabs, plus `current`
- `POST /api/browser/session` — `{"id": "..."}` switches the one current session
- `POST /api/browser/tab` — `{"op": "activate"|"close"|"create", "tab_id": ...}`
- `GET /api/browser/ws` — frames down, gestures and keys up

## Why a headed browser

The screen mirror cannot pass a login wall while the Mac is locked: macOS drops
synthetic HID events and puts the lock screen over the desktop, so the only thing
to mirror is the lock screen itself. Chrome is not affected, because input goes
through the DevTools protocol inside the renderer rather than through the OS
event tap. The window stays headed (not headless) precisely so the page is still
rendered and painted — you click the real pixels, and the QR code that a login
page shows is a real QR code you can scan with the phone in your other hand.

## Capture runs only for viewers

The window and tab exist after the first page load, but nothing is captured, and
no screenshot is taken, until a page opens `/api/browser/ws`. The last viewer
leaving stops the capture loop after a 10 s grace (a tab switch does not pay the
cold start), restores the Mac window size, and leaves the window and its tabs in
place so a return is fast. `GET /api/browser/status` never starts anything.

## Two modes

- **cast** (default) — screenshot polling paced adaptively at ~6 fps idle and
  ~12 fps while input is fresh, deduplicated by an FNV hash of the frame plus a
  1 s keepalive so a static page still proves it is alive.
- **shot** — 1 fps keepalive plus one immediate frame after every interaction.
  Cheapest, good over cellular, and the mode the page falls back to when cast
  fails to produce a frame for 2.5 s (a one-line banner says so).

The choice is remembered in `.ai-critic/browser-config.json`, together with the
preset.

## Sessions and tabs

The left rail lists every session the daemon reports as extension-connected,
plus the current one and any session you opened in this visit; stale registry
rows from earlier runs stay hidden until reopened (`Reopen`). Each row expands to
its tabs, with the driven tab highlighted. Switching sessions is global: every
viewer follows, and capture moves to that session's current tab.

The strip above the stage shows the tabs of the driven window. Clicking one makes
it the capture target, `+` creates a tab (an internal `data:` placeholder page so
the strip never points at something uncapturable), and `×` closes it. Closing the
last content tab is refused with `409` — the session page needs a neighbour, and
a window with no tabs is a dead end. Tabs refresh after every operation and from
a 1.5 s `info` poll while a viewer is attached; the rail itself polls every 8 s.

`+ New window` asks Chrome for a new window (`--new-window <session-url>`), which
is host-side because the extension has no window-creation job.

Tab operations need daemon **1.0.20+**; against an older daemon the page stays
fully watchable and clickable, shows a banner, and those operations fail
read-only with `424` instead of silently doing nothing.

## Window presets

`iphone` (390×844), `ipad` (834×1194) and `mac` (restore the size the window had
before the first shrink). These are true window resizes, not device emulation:
no `Emulation.setDeviceMetricsOverride`, so the page sees a genuinely small
window and its own responsive CSS applies. macOS clamps windows that do not fit
the display, so the server always reports the *achieved* viewport (measured with
`window.innerWidth/innerHeight`) and never the nominal preset. A 390-wide iPhone
window, for example, lands at 500×844 on a 1728 pt display; the phone is told
500 and the cursor math stays correct.

## Gestures and input

Coordinates are normalized to `[0,1]` in the browser and multiplied by the
measured viewport before dispatch, so the client never has to track resizes.

- tap → mouse click at that point
- long press (350 ms) and drag → `mousePressed`, `mouseMoved`, `mouseReleased`;
  the client holds a synthetic cursor at the press point, matching the mirror
- wheel → `Input.dispatchMouseEvent` with `type: mouseWheel`
- keys → `Input.dispatchKeyEvent`, navigation keys (back/forward/reload/home)
  are separate chips
- text → `Input.insertText` with the whole draft in one call

Zoom works in both lenses: one finger pans in VIEW and a double-tap jumps to
2.5×, while pinch zooms and Fit resets. In CTRL a double-click is a real
double-click; a two-finger drag pans and pinch still zooms. Any session or tab
switch starts fitted again.

There is no controller lock: `/browser` is a single-window cockpit, and the
window is only being driven by whoever has the page open. `/screen` keeps its
lock because it drives the real desktop.

## Requirements

- browser-agent daemon **1.0.20 or newer** on the host, with the matching
  extension loaded in Chrome. Load it from the directory the daemon installed,
  `~/.browser-agent/managed-chrome/extensions/browser-agent/<version>`: Chrome
  derives an unpacked extension's id from its absolute path, and the session
  page calls `chrome.runtime.connect()` with the id computed from that same
  path. Loading an arbitrary unzip directory makes the page log
  `Receiving end does not exist` and `/browser` waits for the extension
  forever. Keep the directory stable (a version bump means a new path and a new
  `Load unpacked`).
- A session created by an older daemon keeps the older `extension_install_path`
  in `sessions/<id>/meta.json`; after upgrading the daemon, delete that session
  (`browser-agent session delete <id>`) so it is recreated against the new
  extension path. The extension learns the daemon port from `/go?session=`, so
  the daemon's loopback port is never exposed to the network.
- The page must stay open for capture to run. Backgrounding the tab closes the
  socket; coming back opens a new one and the first frame lands immediately.
- The Mac can be locked, but Chrome must be running with the session window;
  if Chrome quits, the next page load recreates the window.

## Limits

- Zoom is client-side over the latest frame (1×–6×); the page itself is not
  scrolled, so fixed headers stay put.
- Cast is adaptive screenshot polling, not `Page.startScreencast`; there is a
  small capture cost on the host while viewers are connected.
- One driven tab per session at a time. Links that spawn tabs are not followed
  automatically; pick the new tab from the strip. New tabs start on the `data:`
  placeholder, so navigate with the URL bar.
- The window is a normal Chrome window, so extensions and the OS can still draw
  over it; screenshots capture the rendered page, not the whole desktop.

## Testing

`tests/browser-e2e/browser-cast.js` drives the whole path with Playwright: it
launches its own ai-critic binary and Chrome with the unpacked extension, then
asserts frames, input, presets and mode switching end to end. Environment knobs
(`DAEMON`, `APP`, `WWW`, `PROFILE`, `EXT_BUILD`, `SESSION`) let it run beside a
real install without sharing a daemon or a Chrome profile. See
[tests/browser-e2e/README.md](../tests/browser-e2e/README.md).
