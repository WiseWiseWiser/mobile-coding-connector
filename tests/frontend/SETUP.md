# Scenario

**Feature**: frontend smoke and file-transfer tests via Playwright + quick-test

```
# doctest starts isolated quick-test + Vite, then runs Playwright script
doctest Run -> quick-test server + Vite -> BASE_URL

# leaf script.js drives browser; JSON line becomes ScriptResult for Assert
leaf script.js -> Playwright -> ScriptResult -> Assert

# file-transfer leaves may reset/seed {AI_CRITIC_HOME}/file-transfer/ before script
Run -> file-transfer dir (reset/seed) -> FileTransferView + /api/file-transfer
```

## Preconditions

1. The repository root is discoverable (contains `go.mod`).
2. `playwright-debug` is available on `PATH`.
3. Node/npm are available for the Vite dev server started by quick-test.
4. Each test runs with working directory set to its leaf case directory.

## Steps

1. Resolve the repository root and compute a per-test server port (default `3580` plus a hash offset from the test name).
2. Call `lib.QuickTestPrepare` and `lib.QuickTestStart` to build and start the quick-test server and Vite dev server. Unless `Local` mode is enabled, quick-test uses an isolated temp `AI_CRITIC_HOME` with `testpassword` in `server-credentials` (auth is bypassed in quick-test mode). When `Uninitialized` is true, start a normal server without credentials instead.
3. Wait until `/api/quick-test/health` or `/ping` responds on the chosen port.
4. Read the leaf Playwright fixture from `Request.ScriptPath` (default `script.js`, relative to the case directory).
5. Prepend `const BASE_URL = "http://localhost:<port>";` to the script body.
6. Execute the script headlessly by default (`Request.Headless` defaults to `true`):
   - When `Headless=true`: run via `node` using the `playwright-debug` cache
     (`~/.playwright-debug/node_package`) with the Chromium headless-shell channel
     (`chromium.launch({ channel: 'chromium-headless-shell', headless: true })`).
   - When `Headless=false`: run via `playwright-debug run` for visible debugging.
7. Set `CI=true` on the script process environment.
8. Capture stdout/stderr and exit code; parse the last JSON object line into `Response.ScriptResult`.
9. Tear down the quick-test server and Vite processes on cleanup.

## Context

These tests verify frontend navigation smoke scenarios by driving a real browser
against a quick-test server instance. Each leaf supplies a `script.js` fixture
that prints a single JSON line to stdout for machine-readable assertions.

### Parameters (ranked by significance)

| # | Parameter | Type | Values | Description |
|---|-----------|------|--------|-------------|
| 1 | Route (leaf) | path | `/home`, `/home/tools`, `/home/settings`, `/` | Which page or redirect behaviour is exercised (encoded in `script.js`) |
| 2 | `ScriptPath` | string | `script.js` (default) | Playwright fixture filename relative to leaf directory |
| 3 | `ServerPort` | int | 0 → 3580 + hash offset | Quick-test server listen port |
| 4 | `TimeoutSecs` | int | 90–120 | Server readiness and script execution budget |
| 5 | `Headless` | bool | true (default) | Headless shell mode via Playwright; set `false` in leaf `Setup` for visible debugging |
| 6 | `FileTransferReset` | bool | true/false | When true, remove and recreate empty `{AI_CRITIC_HOME}/file-transfer/` before the script |
| 7 | `Uninitialized` | bool | true/false | When true, start normal server without credentials (Setup page tests) |
| 8 | `FileTransferSeeds` | []FileTransferSeed | name + source path | Files copied into `file-transfer/` after server is healthy |

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```