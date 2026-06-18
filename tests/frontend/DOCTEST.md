# Frontend Doctests

Doc-style tests that drive the React UI through Playwright scripts executed via
the `playwright-debug` CLI. Each leaf keeps its browser automation in a
`script.js` fixture alongside `SETUP.md`.

# DSN (Domain Specific Notion)

The frontend doctest harness models a browser-driven smoke-test system for the
ai-critic React app.

**Participants**

- **Doctest runner** — walks the decision tree, chains `Setup` functions,
  calls root `Run`, then leaf `Assert`.
- **Quick-test server** — builds and starts the Go backend on an isolated temp
  `AI_CRITIC_HOME` with test credentials; proxies to Vite.
- **Vite dev server** — serves the React frontend for browser navigation.
- **Playwright** — headless Chromium executes each leaf `script.js`, printing a
  JSON result line to stdout.
- **File Transfer store** — flat `{AI_CRITIC_HOME}/file-transfer/` directory
  backing the dedicated `/api/file-transfer` endpoints and `FileTransferView`.
- **FileTransferView** — WeChat-like inbox: list uploaded files, upload
  (button + drag-and-drop), download, and remove (with confirmation).

**Behaviors**

- Root `Run` starts quick-test, waits for health, optionally seeds the file
  transfer directory, injects `BASE_URL` and `CASE_DIR` into the script, runs
  Playwright, and parses `ScriptResult`.
- Navigation leaves verify routes render expected headings and shell UI.
- File-transfer leaves verify list state, upload, download, and delete against
  the dedicated API and UI.

## Decision Tree

```
[frontend tests]
 |
 +-- navigation/                          (grouping — route smoke tests)
 |    |
 |    +-- home-loads/                     (LEAF)  /home — workspace list
 |    +-- tools-loads/                    (LEAF)  /home/tools — Server Tools
 |    +-- settings-loads/                 (LEAF)  /home/settings — Settings
 |    +-- root-redirects-home/            (LEAF)  / → /home redirect
 |    +-- file-transfer-loads/            (LEAF)  /home/file-transfer — page shell
 |
 +-- file-transfer/                        (grouping — inbox operations)
      |
      +-- list-empty/                     (LEAF)  empty dir → empty-state message
      +-- upload-and-list/                (LEAF)  UI upload → row appears
      +-- download-file/                  (LEAF)  seeded file → browser download
      +-- delete-file/                    (LEAF)  seeded file → remove + API gone
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `navigation/home-loads` | Navigates to `/home` and verifies the workspace list UI renders |
| 2 | `navigation/tools-loads` | Navigates to `/home/tools` and verifies the Server Tools page loads |
| 3 | `navigation/settings-loads` | Navigates to `/home/settings` and verifies the Settings heading is visible |
| 4 | `navigation/root-redirects-home` | Navigates to `/` and verifies redirect to `/home` |
| 5 | `navigation/file-transfer-loads` | Navigates to `/home/file-transfer`; heading and upload area visible |
| 6 | `file-transfer/list-empty` | Empty `file-transfer/` dir shows empty-state message; file count 0 |
| 7 | `file-transfer/upload-and-list` | Upload `testdata/sample.txt` via UI; row with name and size appears |
| 8 | `file-transfer/download-file` | Pre-seeded `hello.txt`; Download triggers save as `hello.txt` |
| 9 | `file-transfer/delete-file` | Pre-seeded `temp.txt`; Remove confirms; row gone and absent from API |

## Parameter Coverage

| Leaf | Route | Storage state | Operation | TimeoutSecs |
|------|-------|---------------|-----------|-------------|
| home-loads | `/home` | — | page load | 90 |
| tools-loads | `/home/tools` | — | page load | 120 |
| settings-loads | `/home/settings` | — | page load | 90 |
| root-redirects-home | `/` | — | redirect | 90 |
| file-transfer-loads | `/home/file-transfer` | any | page load | 90 |
| list-empty | `/home/file-transfer` | empty (reset) | list | 90 |
| upload-and-list | `/home/file-transfer` | empty (reset) | upload + list | 120 |
| download-file | `/home/file-transfer` | seeded `hello.txt` | download | 90 |
| delete-file | `/home/file-transfer` | seeded `temp.txt` | delete | 90 |

## Harness Behaviour (root `Run`)

1. Resolves repo root and starts quick-test server (`lib.QuickTestPrepare` + `lib.QuickTestStart`)
2. Waits for `/api/quick-test/health` or `/ping`
3. Optionally resets/seeds `{AI_CRITIC_HOME}/file-transfer/` per `Request.FileTransferReset` and `Request.FileTransferSeeds`
4. Reads leaf `script.js` from the case directory (working directory at runtime)
5. Injects `const BASE_URL` and `const CASE_DIR` preamble
6. Executes via headless Playwright (default) or `playwright-debug run` when visible
7. Parses the last JSON object line from stdout into `Response.ScriptResult`
8. Tears down quick-test server and Vite on cleanup

## How to Run

Validate tree structure:

```sh
doctest vet ./tests/frontend
```

Run all frontend doctests:

```sh
doctest test ./tests/frontend/...
```

Run navigation leaves:

```sh
doctest test ./tests/frontend/navigation/home-loads
doctest test ./tests/frontend/navigation/file-transfer-loads
```

Run file-transfer leaves:

```sh
doctest test ./tests/frontend/file-transfer/list-empty
doctest test ./tests/frontend/file-transfer/upload-and-list
doctest test ./tests/frontend/file-transfer/download-file
doctest test ./tests/frontend/file-transfer/delete-file
```

Post-implementation verification:

```sh
go run ./script/build
doctest test ./tests/frontend/...
doctest test ./...
```