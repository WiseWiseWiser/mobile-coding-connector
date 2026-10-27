# /browser end-to-end test

One Playwright script drives the whole browser-cast path against real
processes: an ai-critic server, a browser-agent daemon, a Chrome profile with the
unpacked extension (the "Mac" window being driven), and a second Chromium that
pretends to be the iPhone looking at `/browser`.

```sh
cd /tmp/bw-e2e
rm -rf profile-cast profile-cast-chrome
playwright-debug run \
  /Users/xhd2015/Projects/xhd2015/working/ai/external/ai-critic/tests/browser-e2e/browser-cast.js
```

Exit code is 0 only when every check passes; the script prints `PASS` / `FAIL`
lines and an artifact screenshot at `/tmp/bw-e2e/page.png`.

## Why the extension directory matters

Chrome derives an unpacked extension's id from the **absolute directory path**
(`sha256(path)` mapped to `a`–`p`), and the browser-agent session page computes the
id it calls `chrome.runtime.connect(<id>)` with from the session's
`extension_install_path`. Load a different directory and the session page gets
`Could not establish connection. Receiving end does not exist.`, the extension
never reports `connected`, and `/browser` times out with `timeout: extension
connected` — even though the extension's service worker is running.

That is why the script resolves the directory from `GET /v1/sessions` (creating
the session first, 409 being fine) unless `EXT_BUILD` is set explicitly. A stale
session (one created by an older daemon, with an older
`extension_install_path`) reproduces the same failure: give the test daemon a
fresh `--base-dir`, or delete that session, before blaming the extension.

The same rule applies to the real Chrome: load unpacked from
`~/.browser-agent/managed-chrome/extensions/browser-agent/<version>` — the path
the daemon stamps into the session — not from an arbitrary unzip directory. A
packed zip is a byte-for-byte snapshot of that directory apart from the
daemon-generated `bundle-sum.js` and `placeholder.txt`.

## What it proves

- `/browser` serves status and advertises the presets before Chrome exists
- the extension attaches to the session and a content tab appears (a `data:`
  placeholder when the session window only holds `/go?session=`)
- the URL bar navigates the *Chrome* tab, and the page renders there
- a frame reaches the viewer, and a tap at normalized coordinates lands on the
  real button (`#box` turns `rgb(255,136,136)`)
- capture keeps flowing after input, then iPhone/iPad presets resize the real
  window, the achieved viewport matches `window.innerWidth/innerHeight` inside
  Chrome, and the toolbar shows that achieved size
- the Mac preset restores the window size captured before the first shrink
  (asserted against the pre-iPhone viewport, not a hard-coded 1440×900 — a
  Playwright window is not a user's window)
- shot mode keeps a slow keepalive and cast mode can be re-selected

## Environment knobs

| Variable | Default | Meaning |
|----------|---------|---------|
| `EXT_BUILD` | the session's `extension_install_path` | unpacked extension directory passed to `--load-extension`; when unset the script reads it from the daemon (see below) |
| `DAEMON` | `http://127.0.0.1:43799` | browser-agent daemon base URL |
| `APP` | `http://127.0.0.1:18555` | ai-critic server base URL |
| `WWW` | `http://127.0.0.1:18099` | static fixture server (must serve the spike page) |
| `SESSION` | `ai-critic` | session id used by `/go?session=` |
| `PROFILE` | `/tmp/bw-e2e/profile-cast` | Chromium profile for the viewer; the driven Chrome uses `$PROFILE-chrome` |

## Prerequisites

1. **Static fixture** — `python3 -m http.server 18099` in a directory whose
   `index.html` has `#btn` (turns `#box` pink) and `#box`.
2. **Isolated daemon** — never point the test at the production daemon on 43761:

   ```sh
   browser-agent serve --port 43799 --base-dir /tmp/bw-e2e/base
   ```

3. **ai-critic dev server** — built with `npm run build` first (the React dist is
   embedded), started without auth and pointed at the test daemon:

   ```sh
   BROWSER_AGENT_PORT=43799 BROWSER_AGENT_NO_OPEN=1 \
     go run ./script/server/build -o /tmp/ai-critic-dev
   BROWSER_AGENT_PORT=43799 BROWSER_AGENT_NO_OPEN=1 \
     /tmp/ai-critic-dev --port 18555 --quick-test --keep --no-event-bus-publish
   ```

   `BROWSER_AGENT_NO_OPEN=1` keeps the manager from opening a real Chrome
   window outside the test profile. `--quick-test` disables auth so the script
   can poll `/api/browser/status`.

4. **`ws` for the raw socket watcher** — the script requires
   `/tmp/wsprobe/node_modules/ws` (or adjust the `require` path).

## Hygiene

- The test writes only under `/tmp/bw-e2e` and its own profiles; it must not
  resize or drive a real user window.
- Kill the test daemon, fixture server and dev server when finished; the daemon
  listens on loopback only and never on the production port.
- If Chrome quits mid-run, delete `$PROFILE-chrome` before retrying, otherwise a
  stale extension registration can win.

## Reading a failure

The script prints `status at failure:` and the last socket messages. A status
with `ok:false` and `error` usually means the daemon or extension is missing; a
`timeout: content tab in Chrome` means the extension did not adopt the session
page or a `data:` placeholder was rejected. Check the daemon log and
`/api/browser/status` by hand before touching the script.
