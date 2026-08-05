# Scenario

**Feature**: Full stack ServeService Dial through agent tunnel + CryptoSSHRunner

```
# production-shaped compose (BackendDial nil → AdhocServer)
CreateSSHSession(pubkey) -> Adhoc authorized
ServeService{Dial: SSHTunnelDialFunc}.Start
CryptoSSHRunner.Run(echo p4-ok) -> stdout contains p4-ok
```

## Preconditions

- Scenario family: through-serve.
- Uses P2 ServeService + P3 CryptoSSHRunner/Adhoc + P4 client tunnel.

## Steps

1. CreateSession with client pubkey (Manager starts Adhoc).
2. Start ServeService with client DialFunc.
3. Run remote command via CryptoSSHRunner on session LocalPort.

## Context

- Exit criterion for P4 L2: SSH command through agent WS tunnel.

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
	if len(req.RemoteArgv) == 0 {
		req.RemoteArgv = []string{"echo", "p4-ok"}
	}
	if req.EchoNeedle == "" {
		req.EchoNeedle = "p4-ok"
	}
	return nil
}
```
