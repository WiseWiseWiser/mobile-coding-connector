# Scenario

**Feature**: dot-pkgs ptywrap client talks to fake `/api/exec/ws` server

```
# interactive exec client
remoteexec/client.RunInteractive -> dial /api/exec/ws -> fake server messages
```

## Preconditions

- `github.com/xhd2015/agent-pro/pkgs/remoteexec/client` implements `RunInteractive`
  with `SkipTTYCheck` for doctests.
- Fake server uses httptest + gorilla/websocket at `/api/exec/ws`.

## Steps

1. Leaf sets `req.Phase` and WS scenario fields.
2. Harness starts fake server and runs `Client.RunInteractive`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.WSExecArgv) == 0 {
		req.WSExecArgv = []string{"true"}
	}
	return nil
}
```