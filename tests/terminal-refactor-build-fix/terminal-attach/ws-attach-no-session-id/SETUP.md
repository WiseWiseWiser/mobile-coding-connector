# Scenario

**Feature**: dot-pkgs ptywrap client attach must not panic when the server stays silent

```
# attach handshake path
client.AttachWithIO -> dial /api/terminal -> readSessionID(conn)
server upgrades WS but never sends {"type":"session_id",...}
```

**Bug**: `readSessionID` (`dot-pkgs/go-pkgs/shell/ptywrap/client/attach.go:162`)
loops calling `conn.ReadMessage()` with a 2s read deadline. When the deadline
expires, gorilla marks the connection's `readErr` permanently
(`hideTempErr` wraps the timeout as a `*netError` with `Timeout()==true` but
`Temporary()==false`). The loop's `ne.Timeout()` branch then `continue`s, but
every subsequent `ReadMessage` returns immediately from the cached `readErr`
(the 2s deadline no longer applies), so `readErrCount` climbs to 1000 and gorilla
panics:

```
panic: repeated read on failed websocket connection
github.com/gorilla/websocket.(*Conn).NextReader ... conn.go:1030
...client.readSessionID ... attach.go:166
```

Triggered in production by `local-agent terminal attach session-5` against a
daemon that accepts the WebSocket but never delivers a `session_id` frame.

## Preconditions

- `github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client` exports
  `AttachWithIO`, `Client`, `ConnectOptions` (with `SkipTTYCheck`/`Wait`).
- Fake server uses httptest + gorilla/websocket at `/api/terminal`.

## Steps

1. Leaf sets `req.Phase = "ws-attach-no-session-id"`.
2. Harness starts a fake `/api/terminal` server that upgrades the WS and then
   holds the connection open without sending any message.
3. Harness runs `AttachWithIO` in a recovering goroutine so a panic is captured
   as `resp.AttachPaniced` instead of crashing the test process.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "ws-attach-no-session-id"
	req.WSAttachSilent = true
	return nil
}
```
