# Scenario

**Bug**: refactored server panics on boot — duplicate `/api/terminal` route

```
# server boot (mirrors `ai-critic-server keep-alive` -> spawn child)
server.Serve -> server.RegisterAPI(mux) -> terminal.RegisterAPI(mux)
# terminal.RegisterAPI double-registers /api/terminal:
#   ptywrap.RegisterAPIWithManager(mux, mgr)  -> mux.HandleFunc("/api/terminal", ...)   (handler.go:26)
#   mux.HandleFunc("/api/terminal", sshWrapper)                                       (terminal.go:60)
# Go 1.22+ ServeMux panics on the conflicting duplicate pattern
```

**Root cause**: commit `0f398d8` ("refactor terminal as thin ptywrap adapter")
made `terminal.RegisterAPI` call `ptywrap.RegisterAPIWithManager`, which already
registers `/api/terminal` (plus `/api/terminal/sessions` and
`/api/terminal/sessions/`). The adapter then registers `/api/terminal` **again**
to layer the SSH wrapper on top. Go 1.22+'s pattern-based `ServeMux` treats this
as a conflicting duplicate and panics on the second registration — so the server
process dies immediately at boot:

```
panic: pattern "/api/terminal" (registered at server/terminal/terminal.go:60)
conflicts with pattern "/api/terminal" (registered at .../ptywrap/handler.go:26):
/api/terminal matches the same requests as /api/terminal
net/http.(*ServeMux).register(...)
github.com/xhd2015/ai-critic/server/terminal.RegisterAPI  terminal.go:60
github.com/xhd2015/ai-critic/server.RegisterAPI            server.go:455
github.com/xhd2015/ai-critic/server.Serve                  server.go:228
```

This is why a rebuilt server never actually starts: the keep-alive supervisor
respawns the child, it panics before binding the port, and the port stays
"connection refused" forever. The previously-running (pre-refactor) daemon
kept serving from memory — which is also why `terminal attach` still timed out
against stale in-memory sessions (see `terminal-attach/ws-attach-existing-session`).

`go build ./` passes (the existing `compile/server-builds` leaf is GREEN), so
this is a **runtime** regression not caught by compile-only checks.

## Preconditions

- `github.com/xhd2015/ai-critic/server/terminal` exports `RegisterAPI(mux *http.ServeMux)`.
- Go 1.22+ `net/http.ServeMux` panics on duplicate/conflicting pattern registration.

## Steps

1. Leaf sets `req.Phase = "server-api-register"`.
2. Harness creates a fresh `http.NewServeMux()` and calls `terminal.RegisterAPI`
   inside a `recover`, capturing any panic as `resp.RegisterPaniced` /
   `resp.RegisterPanic` so the test process is not killed.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "server-api-register"
	return nil
}
```
