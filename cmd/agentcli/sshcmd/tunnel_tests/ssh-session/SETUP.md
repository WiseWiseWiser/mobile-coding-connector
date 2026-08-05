# Scenario

**Feature**: HTTP CreateSSHSession against sshtunnel Manager

```
# authorized vs unauthorized Bearer
Client.CreateSSHSession -> POST /api/remote-agent/ssh/sessions
  authorized -> session_id
  unauthorized -> error
```

## Preconditions

- Scenario family: session CreateSession.
- Manager.RequiredToken set; client Token varies by leaf.

## Steps

1. Start httptest with RegisterAPIWithManager.
2. Call CreateSSHSession; record SessionID / CreateErr.

## Context

- Auth is package-level RequiredToken for L2 (production uses server middleware).

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
	return nil
}
```
