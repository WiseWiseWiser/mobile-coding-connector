# Screen Mirror

`/screen` mirrors the host desktop into the browser and, once you take control,
drives the real mouse and keyboard. It is the remote-control path: open it on a
phone through the tunnel and the Mac is usable from there.

- `GET /screen` — the page (React, embedded in the binary)
- `GET /api/screen/status` — platform, permissions, whether input is available
- `GET /api/screen/ws` — the stream and the input channel

Viewing needs nothing but the page. Input needs Accessibility (macOS) and is
granted to the AI Critic app, not to the browser.

## When capture runs

Nothing is captured until a page asks for it. Opening `/screen` is the ask: the
page opens the WebSocket, and the server starts the capture child (the
ScreenCaptureKit helper plus the `caffeinate` assertion) on the first viewer and
stops it when the last one is gone. `GET /api/screen/status` never starts
anything; the status only reports what is already running. Idle cost is zero:
no process, no recording, no power assertion.

A hidden page is not a viewer. Backgrounding the tab, locking the phone or
switching apps closes the socket; coming back opens a new one, and the canvas
keeps the last frame until the fresh one lands. Closing the socket also releases
control, so a return always lands read-only and **Take control** has to be pressed
again.

The stop waits out a 10 s grace period, so a quick tab switch does not pay the
~0.8 s cold start: inside the window the stream stays warm and a return is live
almost immediately. After it the helper is stopped and the display may sleep
again.

## Taking control

Only one client drives the host at a time.

1. Open `/screen`; the page connects and shows the live picture, read-only.
2. Press **Take control** in the top bar. The bar turns green and the on-screen
   key bar becomes live.
3. Press **Release** (or close the tab) to hand it back.

A tap on the picture does **not** take control — in WEB mode it pans. Use the
button, or switch to REMOTE first.

## Two modes

The `REMOTE` / `WEB` chip in the top bar and in key bar row 1 switches what a
gesture means. The chip shows the mode you are *in*, not the one you would
switch to.

| | REMOTE | WEB |
|---|---|---|
| One-finger drag | moves the host cursor | pans the picture |
| Long press, then drag | drags (window move, selection) | nothing |
| Two-finger drag | scrolls on the host | pinch zoom |
| Tap | left click | nothing |
| Double tap | double click | toggles 1× ⇄ 2.5× |
| Key bar | live | host keys dimmed; the text field stays editable |
| Host cursor overlay | drawn | hidden |
| `− ＋ ⤢` | works | works |

WEB mode sends **no frames at all**: no pointer, no key, no wheel. It is safe to
poke at while someone else — or you — is driving the machine.

Switching to REMOTE does not by itself take control; the lock is unchanged, and
the server's idle timeout may have dropped it in the meantime, in which case
press **Take control** again.

## Typing text

The text field in the key bar can be filled in from either mode, so a sentence
can be composed while browsing a zoomed-in area. Sending is stricter than
composing: **send needs control on the host and REMOTE mode**. In WEB, or while
view-only, the send button is disabled and an amber note under the row says
`draft — take control to send`; the placeholder reads the same way.

A draft is never thrown away by a failed send:

- Pressing Enter with nothing to send leaves the text alone — it does not fire
  and does not clear the field.
- If the server does refuse an input (`not_controller`, `host_locked`,
  `input_failed`), the text that was in the field is put back, provided you have
  not edited it in the meantime and the refusal arrives within 3 s.
- Escape drops focus in WEB and keeps both the draft and the key bar; in REMOTE
  it clears the field and closes the bar.
- Switching modes keeps the draft.

The refusal path is a safety net, not the normal way to learn that a send is
impossible: the server sends at most one `not_controller` notice per second per
viewer, so a second attempt inside that window fails silently. That is why the
button is disabled up front instead of letting you press it and wait for an
error.

## Zoom and pan

- `＋` / `−` step by 1.5× anchored at the centre of the picture, clamped to
  1×–6×.
- Pinch zooms about the midpoint of the two fingers; the pixel under the fingers
  stays put.
- One finger (WEB) pans. Panning stops at the edges of the image; an axis that
  already fits is centred and does not pan.
- `⤢` returns to the fit-to-stage 1× framing. It is disabled at 1×.
- Desktop browsers can use the wheel to zoom.

The zoom is digital and client-side: the stream stays at `max_width` (1280 by
default), so past ~2× an iPhone shows a magnified 1280-wide stream rather than a
finer one. Text stays readable because the glyphs were already sampled; it just
gets soft at the top end. A crisper maximum needs a higher `max_width` or a
server-side region-of-interest crop, which is not implemented.

## Dragging

A finger on glass cannot both move a pointer and hold a button, so a drag is
latched by time instead of by a second button:

1. In REMOTE mode, with control, put one finger down and **keep it still for
   350 ms**. The frame turns amber and a `DRAG · lift to drop` pill appears at
   the top of the picture. Nothing has been pressed on the host yet.
2. Slide. The pointer moves with the finger and the host sees a held-button
   drag, so windows move, text selects, sliders slide.
3. Lift. The button is released and the amber frame clears. A lifted drag never
   becomes a click.

The latch is deliberately easy to lose:

- Travelling more than 12 px before the 350 ms are up cancels it, so a normal
  one-finger swipe still just moves the pointer.
- A second finger cancels it, and so does switching to WEB.
- Losing control (another viewer, the idle timeout, **Release**) drops it
  locally without sending a release: the server already let go with the lock.
- There are no haptics; Safari does not expose `navigator.vibrate`.

Desktop browsers need none of this: pressing and moving with the mouse already
sends `leftMouseDragged`, and no badge appears.

The known rough edge: the latch needs 12 px of stillness, and while zoomed in one
image pixel is several screen pixels, so a very slow, very short drag can be
rejected as motion. Drag from a comfortable spot, or drop back to 1× first.

## Fullscreen

`⛶` requests element fullscreen where the browser has it. iOS Safari only
fullscreens `<video>`, so there the page switches to an **immersive** layout
instead: the browser chrome is dropped as far as it can be, the top bar is
replaced by a thin `‹ chip view only ✕` bar over the picture, and the key bar
stays. Escape, the back arrow, or `✕` leaves it.

An iPhone can hide the browser bars for good by installing the page: Share →
Add to Home Screen. The manifest is already `display: standalone`, so the
installed app opens without Safari chrome. The page shows that hint once in
immersive mode.

## Limits worth knowing

- Protected windows (banking, DRM, some system dialogs) capture as black.
- Nobody watching ⇒ no capture: the host screen is not recorded when no page is
  open, and the display is allowed to sleep again.
- While the host is locked the mirror shows the login window and input is paused.
- Dragging a window smears until the next dirty tile arrives.
- The stream is ~12 fps, so motion is choppy; input round-trip is ~85–330 ms.
- macOS 26 dropped the legacy capture API; a small Swift helper using
  ScreenCaptureKit does the capture. The server holds a `caffeinate` assertion
  whenever capture is running, because a sleeping display has nothing to send.
- Wayland is refused; X11 input is best-effort.

## Implementation map

| Path | What it holds |
|---|---|
| `server/screen/` | capture, tiling, WebSocket hub, input injection, control lock |
| `server/screen/screencap/` | the ScreenCaptureKit helper |
| `ai-critic-react/src/v2/mcc/screen/ScreenPage.tsx` | page state: control, fullscreen, keys, mode |
| `…/screen/RemoteScreen.ts` | socket, canvas blit, gestures, coordinate mapping |
| `…/screen/viewport.ts` | pure zoom/pan geometry (unit-tested) |
| `…/screen/MirrorView.tsx` | pointer routing for the two modes, drag latch wiring |
| `…/screen/screenDrag.ts` | the pure long-press drag latch (unit-tested) |
| `…/screen/KeyBar.tsx` | on-screen keyboard, composer, mode chip |
| `…/screen/screenInput.ts` | pure send/placeholder rules (unit-tested) |
| `…/screen/screen.css` | layout, immersive layout, zoom cluster |

All viewport geometry is in CSS pixels, because the canvas context is scaled by
`devicePixelRatio` before drawing; adding a dpr factor would double-apply it.
