# Scenario

**Feature**: client.SSHTunnelDial duplex WebSocket tunnel

```
# binary path
CreateSession -> SSHTunnelDial -> net.Conn write/read

# lifecycle
DestroySession -> SSHTunnelDial fails
```

## Preconditions

- Scenario family: tunnel.
- binary-echo uses Manager.BackendDial → TCP echo (isolates splice from SSH).
- after-destroy uses same harness then DestroySSHSession.

## Steps

1. CreateSession via authorized client.
2. Dial or destroy+dial per leaf.

## Context

- WS path: GET `/api/remote-agent/ssh/sessions/{id}/tunnel`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	_ = req
	if req.ManagerToken == "" {
		req.ManagerToken = "test-token"
	}
	if req.Token == "" {
		req.Token = "test-token"
	}
	return nil
}
```
