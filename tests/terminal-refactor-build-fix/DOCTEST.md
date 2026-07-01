# Terminal Refactor Build-Fix Doctests

Regression tests for restoring green builds after the terminal ptywrap adapter
refactor (`0f398d8`). Covers compile recovery, `ptywrap.ShellQuote` migration,
and interactive `/api/exec/ws` client behavior in `dot-pkgs/shell/ptywrap/client`.

# DSN (Domain Specific Notion)

**Participants**

- **ai-critic module** — `go build ./cmd/remote-agent` and `go build ./` must succeed
  once deleted helpers are wired to the shared `dot-pkgs` core.
- **dot-pkgs `shell/ptywrap`** — exports `ShellQuote`; consumed by
  `run/keep_alive.go` and `run/rebuild.go`.
- **dot-pkgs `shell/ptywrap/client`** — interactive WebSocket exec client
  (`RunExec`/`ExecOptions`) for `/api/exec/ws`; owns the WS bridge helpers
  removed from `cmd/agentcli/bash.go` and `exec.go`.
- **Fake exec WS server** — httptest handler upgrades `/api/exec/ws`, accepts
  `ExecRequest`, emits binary stdout and control JSON (`exit`, `error`).
- **POSIX `sh`** — validates shell-quote round-trips and keep-alive script syntax.

**Behaviors**

- Broken compile targets become buildable once ai-critic imports the `dot-pkgs`
  core directly (no restored inline WS helpers in `bash.go`/`exec.go`).
- `ShellQuote` produces injection-safe tokens embeddable in `sh -c` scripts.
- `outputKeepAliveScript` embeds quoted paths when bin/args contain spaces; `sh -n` passes.
- `RunExec` mirrors pre-refactor exec.go: captures stdout, returns remote exit
  code; surfaces server `error` messages and HTTP dial failures with status + body.
- `AttachWithIO` performs the `/api/terminal` `session_id` handshake without
  panicking when the server stays silent: a poisoned read deadline must surface
  as a graceful timeout error, not gorilla's `repeated read on failed websocket
  connection` panic.
- `AttachWithIO` stays backward-compatible with pre-refactor daemons that omit
  the `session_id` echo on reattach: when the caller supplies a `SessionID` and
  the server begins serving, attach proceeds with the known SessionID instead
  of timing out.
- `terminal.RegisterAPI` registers all terminal routes on a fresh `ServeMux`
  without panicking: the adapter must not double-register `/api/terminal` on top
  of `ptywrap.RegisterAPIWithManager`, or the server panics at boot under Go
  1.22+.

## Version

0.0.2

## Decision Tree

```
[post-terminal-refactor build fix]
 |
 +-- compile/                              (GROUP) go build must succeed
 |    +-- remote-agent-builds/              (LEAF) go build ./cmd/remote-agent
 |    +-- server-builds/                    (LEAF) go build ./
 |
 +-- shell-quote/                           (GROUP) ptywrap ShellQuote + keep-alive embed
 |    +-- simple-path-unquoted-safe/        (LEAF) ShellQuote simple path round-trips in sh
 |    +-- spaces-and-special-chars/         (LEAF) spaces + apostrophe safe quoting
 |    +-- keep-alive-script-embeds-quote/   (LEAF) outputKeepAliveScript quotes spaced paths
 |
 +-- remoteexec-client/                     (GROUP) fake /api/exec/ws server
 |    +-- ws-exec-exit-code/                (LEAF) binary stdout + exit code 42
 |    +-- ws-exec-error-message/            (LEAF) {"type":"error","message":"boom"}
 |    +-- ws-dial-http-error/               (LEAF) HTTP 401 + JSON body in dial error
 |
 +-- terminal-attach/                       (GROUP) fake /api/terminal attach handshake
 |    +-- ws-attach-no-session-id/          (LEAF) silent server -> graceful timeout, no panic
 |    +-- ws-attach-existing-session/       (LEAF) stale-daemon reattach (no session_id echo) -> proceed, no timeout
 |
 +-- server-boot/                           (GROUP) server boots / routes register without panic
      +-- api-register-no-panic/            (LEAF) terminal.RegisterAPI must not double-register /api/terminal
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `compile/remote-agent-builds` | `go build -o /dev/null ./cmd/remote-agent` exits 0 |
| 2 | `compile/server-builds` | `go build -o /dev/null ./` exits 0 |
| 3 | `shell-quote/simple-path-unquoted-safe` | `ShellQuote("/tmp/ai-critic")` round-trips via `sh -c` |
| 4 | `shell-quote/spaces-and-special-chars` | `ShellQuote` for spaced args and `it's` |
| 5 | `shell-quote/keep-alive-script-embeds-quote` | keep-alive script quotes spaced bin/args; `sh -n` OK |
| 6 | `remoteexec-client/ws-exec-exit-code` | fake WS returns stdout + exit 42 |
| 7 | `remoteexec-client/ws-exec-error-message` | fake WS error message surfaces in client err |
| 8 | `remoteexec-client/ws-dial-http-error` | HTTP 401 body snippet in dial error |
| 9 | `terminal-attach/ws-attach-no-session-id` | silent `/api/terminal` server -> graceful timeout, no panic |
| 10 | `terminal-attach/ws-attach-existing-session` | stale-daemon reattach (no session_id echo) -> proceed with known SessionID, no timeout |
| 11 | `server-boot/api-register-no-panic` | `terminal.RegisterAPI` must not double-register `/api/terminal` (server boot panic) |

## Parameter Coverage

| Factor (significance →) | Leaves |
|-------------------------|--------|
| Verification mode (compile vs unit vs WS fake vs boot smoke) | compile/*, shell-quote/*, remoteexec-client/*, terminal-attach/*, server-boot/* |
| Build target | remote-agent-builds, server-builds |
| ShellQuote input shape | simple-path, spaces-and-special-chars, keep-alive embed |
| WS server outcome | exit code, error message, HTTP dial failure, silent attach (no session_id) |
| Server boot | route registration must not double-register /api/terminal |

## How to Run

```sh
doctest vet ./tests/terminal-refactor-build-fix
doctest test ./tests/terminal-refactor-build-fix/...
```

Post-implement verify:

```sh
doctest test ./tests/terminal-refactor-build-fix/...
go run ./script/build
```

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/tests/terminal-refactor-build-fix/testdata/buildfixtest"
)

type Request = buildfixtest.Request
type Response = buildfixtest.Response

func Run(t *testing.T, req *Request) (*Response, error) {
	return buildfixtest.Run(t, req)
}
```