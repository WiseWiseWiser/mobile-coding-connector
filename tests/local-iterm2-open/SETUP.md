# Scenario

**Feature**: local menu bar opens directories in iTerm2 via POST /api/local/iterm2/open

```
# client (out of tree) -> Bearer POST /api/local/iterm2/open
# server handler -> ParseOpenMode + iterm2.OpenConfig(dir, cfg)
# inject Open / Osascript -> no live iTerm, no kool binary

# auth middleware wraps mux; open path is NOT skip-listed
Authorization Bearer -> handler; missing -> 401
```

## Preconditions

1. Package `github.com/xhd2015/ai-critic/server/localiterm2` exports:
   - `ParseOpenMode(s string) (iterm2.OpenMode, error)`
   - `Handler` with injectible `Open func(dir string, cfg *iterm2.Config) error`
   - `(*Handler).ServeHTTP` for the open endpoint
   - `Register(mux *http.ServeMux, h *Handler)` mounting `POST /api/local/iterm2/open`
2. Tree stays **RED** until that package and `server.Serve` registration exist
   (compile failure or assert failure).
3. No live iTerm: success leaves inject `Open` (optionally call real
   `iterm2.OpenConfig` with `Installed`/`Osascript` hooks and `SetGOOSForTest`).
4. Auth leaves use a temp credentials file via `auth.SetCredentialsFile`.
5. Do **not** exec the `kool` binary.

## Steps

1. Root `Setup` validates request pointer.
2. Grouping `Setup` sets `Op`.
3. Leaf `Setup` sets mode/dir/send/auth fields and temp paths.
4. Root `Run` dispatches parse / httptest open / register / auth / skip_list.
5. Leaf `Assert` checks mode, status, error envelope, or injection records.

## Context

Implements REQUIREMENT-DESIGN-local-iterm2-open.md (server/pure). Swift client
contracts live in `tests/macos-menubar-projects` and
`tests/macos-menubar-terminals`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req == nil {
		t.Fatal("nil request")
	}
	return nil
}
```
