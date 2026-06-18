# Frontend Doctests

Doc-style tests that drive the React UI through Playwright scripts executed via
the `playwright-debug` CLI. Each leaf keeps its browser automation in a
`script.js` fixture alongside `SETUP.md`.

## Decision Tree

```
[frontend smoke tests]
 |
 +-- navigation/
      |
      +-- home-loads/          (LEAF)  /home — workspace list renders
      +-- tools-loads/         (LEAF)  /home/tools — Server Tools heading or Foundation category
      +-- settings-loads/      (LEAF)  /home/settings — Settings heading visible
      +-- root-redirects-home/ (LEAF)  / → redirects to /home
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `navigation/home-loads` | Navigates to `/home` and verifies the workspace list UI renders (`Your Projects` heading, `.mcc-workspace-list`) |
| 2 | `navigation/tools-loads` | Navigates to `/home/tools` and verifies the Server Tools page loads (`h2` or `Foundation` category label) |
| 3 | `navigation/settings-loads` | Navigates to `/home/settings` and verifies the Settings heading is visible |
| 4 | `navigation/root-redirects-home` | Navigates to `/` and verifies the app redirects to `/home` |

## Parameter Coverage

| Leaf | Route | ScriptPath | TimeoutSecs | ServerPort |
|------|-------|------------|-------------|------------|
| home-loads | `/home` | `script.js` | 90 | 3580 + hash offset |
| tools-loads | `/home/tools` | `script.js` | 120 | 3580 + hash offset |
| settings-loads | `/home/settings` | `script.js` | 90 | 3580 + hash offset |
| root-redirects-home | `/` | `script.js` | 90 | 3580 + hash offset |

## Harness Behaviour (root `Run`)

1. Resolves repo root and starts quick-test server (`lib.QuickTestPrepare` + `lib.QuickTestStart`)
2. Waits for `/api/quick-test/health` or `/ping`
3. Reads leaf `script.js` from the case directory (working directory at runtime)
4. Injects `const BASE_URL = "http://localhost:<port>";` preamble
5. Executes via `playwright-debug run '<script>'`
6. Parses the last JSON object line from stdout into `Response.ScriptResult`
7. Tears down quick-test server and Vite on cleanup

## How to Run

Validate tree structure:

```sh
doctest vet ./tests/frontend
```

Run all frontend doctests:

```sh
doctest test ./tests/frontend/...
```

Run a single leaf:

```sh
doctest test ./tests/frontend/navigation/home-loads
doctest test ./tests/frontend/navigation/tools-loads
doctest test ./tests/frontend/navigation/settings-loads
doctest test ./tests/frontend/navigation/root-redirects-home
```

Post-implementation verification:

```sh
go run ./script/build
doctest test ./tests/frontend/...
doctest test ./...
```