# Scenario

**Bug**: rebuilt client times out attaching to a running session held by a
stale (pre-refactor) daemon

```
# attach to a live session held by a pre-refactor daemon
client.AttachWithIO(SessionID="session-5") -> dial /api/terminal?session_id=session-5
# pre-refactor HandleTerminalWebSocket: sessionID != "" && s != nil -> reattach
#   falls through to s.attach(conn) WITHOUT sending {"type":"session_id",...}
server -> binary output frame (session is live, streaming)   <- no session_id echo
client.readSessionID -> demands session_id frame -> 10s deadline -> timeout
```

**Root cause**: the pre-refactor `handleTerminalWebSocket`
(`server/terminal/terminal.go` at `0f398d8^`) only emitted a `session_id`
frame on the **create** paths (`s == nil`: SSH create and new-shell create).
On **reattach** (`sessionID != ""` and `s != nil`) it fell straight through to
`s.attach(conn, attachMode)` and never sent `session_id` — the old client did
not require it on attach.

The new dot-pkgs client (`readSessionID` in
`dot-pkgs/go-pkgs/shell/ptywrap/client/attach.go`) unconditionally demands a
`session_id` frame on every connection, including attach. So a freshly rebuilt
client attaching to `session-5` (still held in memory by a running
pre-refactor daemon) never receives `session_id` and times out:

```
$ local-agent terminal attach session-5
Error: timeout waiting for session_id message
```

`terminal list` shows the session as `running` because the old daemon still
owns it. Restarting the daemon clears it (and would also lose the session),
but the robust fix is client-side: when the caller already supplied a
`SessionID` (attach mode) and the server begins serving without echoing
`session_id`, the client should proceed with the known SessionID instead of
timing out — restoring backward compatibility with pre-refactor daemons.

## Preconditions

- `github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client` exports
  `AttachWithIO`, `Client`, `ConnectOptions` (with `SkipTTYCheck`/`Wait`).
- Fake server uses httptest + gorilla/websocket at `/api/terminal`.

## Steps

1. Leaf sets `req.Phase = "ws-attach-existing-session"` and
   `req.AttachKnownSessionID = "session-5"`.
2. Harness starts a fake `/api/terminal` server that mimics the pre-refactor
   reattach: upgrade the WS, immediately write a binary output frame (the
   session is live and streaming), and never send a `session_id` frame.
3. Harness runs `AttachWithIO` with `SessionID="session-5"`,
   `AttachSnapshot: true`, `Wait: true` in a recovering goroutine.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "ws-attach-existing-session"
	req.AttachKnownSessionID = "session-5"
	return nil
}
```
