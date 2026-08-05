# Scenario

**Feature**: remote-agent ssh P4 — agent duplex tunnel (session + WS Dial + serve)

```
# create remote session (Adhoc or test backend)
Client.CreateSSHSession + Bearer -> sshtunnel.Manager -> session_id

# duplex tunnel as net.Conn
Client.SSHTunnelDial(session_id) -> WS binary -> TCP backend/Adhoc

# production serve compose
ServeService{Dial: SSHTunnelDialFunc} -> LocalRelay
CryptoSSHRunner -> LocalPort -> Dial -> remote echo p4-ok

# agentcli wire
BuildSSHTunnelDial(Client) -> non-nil DialFunc for --serve
```

## Preconditions

- Packages (RED until implementer):
  - `github.com/xhd2015/ai-critic/server/sshtunnel`
  - `client.CreateSSHSession`, `DestroySSHSession`, `SSHTunnelDial`, `SSHTunnelDialFunc`
  - `agentcli.BuildSSHTunnelDial`
- Existing GREEN (do not modify): P1 `tests/`, P2 `session_tests/`, P3 `ssh_tests/`.
- Parallel-safe: absolute paths from `d.DOCTEST_CASE`; httptest per leaf; ports via
  Listen `:0`; no `Setenv`/`Chdir`; no process-global stdout reassignment.
- Auth in L2: `Manager.RequiredToken` + `client.Token` (not full `server.Serve` bootstrap).
- gorilla/websocket already in go.mod (same as `/api/exec/ws`).

## Steps

1. Root Setup defaults ProfileID, Root, ConfigDir, Token, EchoPayload, RemoteArgv, EchoNeedle.
2. Grouping Setup sets Scenario family.
3. Leaf Setup sets concrete Scenario and auth/payload overrides.
4. Root `Run` dispatches on Scenario and records Response fields.

## Context

- Module: `github.com/xhd2015/ai-critic`
- Tree home: `cmd/agentcli/sshcmd/tunnel_tests` (sibling of P1–P3)
- Layer: L2 in-process httptest only; no L3 product-binary e2e

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req.ProfileID == "" {
		req.ProfileID = "default"
	}
	if req.Root == "" {
		req.Root = filepath.Join(d.DOCTEST_CASE, "store-root")
	}
	if req.ConfigDir == "" {
		req.ConfigDir = filepath.Join(d.DOCTEST_CASE, "session-config")
	}
	if req.Token == "" {
		req.Token = "test-token"
	}
	if req.EchoPayload == "" {
		req.EchoPayload = "p4-tunnel-hi"
	}
	if len(req.RemoteArgv) == 0 {
		req.RemoteArgv = []string{"echo", "p4-ok"}
	}
	if req.EchoNeedle == "" {
		req.EchoNeedle = "p4-ok"
	}
	return nil
}
```
