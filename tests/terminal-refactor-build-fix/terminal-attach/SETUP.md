# Scenario

**Feature**: dot-pkgs ptywrap client attach handshake over `/api/terminal`

```
# interactive attach client
ptywrap/client.AttachWithIO -> dial /api/terminal -> readSessionID handshake
```

## Preconditions

- `github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client` implements
  `AttachWithIO` with `SkipTTYCheck` for doctests.
- Fake server uses httptest + gorilla/websocket at `/api/terminal`.

## Steps

1. Leaf sets `req.Phase` and attach-scenario fields.
2. Harness starts fake `/api/terminal` server and runs `AttachWithIO` in a
   recovering goroutine so panics are captured as response fields.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Phase == "" {
		req.Phase = "ws-attach-no-session-id"
	}
	return nil
}
```
