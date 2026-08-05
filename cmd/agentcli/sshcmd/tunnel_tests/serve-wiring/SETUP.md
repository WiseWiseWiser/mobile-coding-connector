# Scenario

**Feature**: agentcli serve wiring builds Dial from Client

```
# --serve starter contract
BuildSSHTunnelDial(Client, publicKey) -> non-nil DialFunc + session info
```

## Preconditions

- Scenario family: serve-wiring.
- Does not require full ServeService.Start (unit surface for Dial assignment).

## Steps

1. Start httptest sshtunnel; authorized client.
2. Call agentcli.BuildSSHTunnelDial.
3. Assert Dial non-nil; optional smoke dial succeeds.

## Context

- Production: sshServeStarter uses this when Client is configured (closes P3 Dial=nil gap).

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
