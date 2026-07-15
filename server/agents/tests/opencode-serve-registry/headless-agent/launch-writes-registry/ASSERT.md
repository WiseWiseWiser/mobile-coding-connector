---
label: slow
explanation: real opencode serve startup when fake not forced; fake path is fast
---

## Expected

1. `LaunchErr` is nil.
2. `RegistryErr` is nil.
3. At least one registry child with `kind=headless-agent`, `agent_id=grok`.
4. Registry `pid` matches listener on `LaunchSession.Port` (via lsof or export).
5. Registry `port` equals `LaunchSession.Port`.

## Errors

- Missing registry file or mismatched pid/port fails the test.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.LaunchErr != nil {
		t.Fatalf("launch error: %v", resp.LaunchErr)
	}
	if resp.RegistryErr != nil {
		t.Fatalf("read registry error: %v", resp.RegistryErr)
	}
	if resp.LaunchSession == nil {
		t.Fatal("expected launch session")
	}
	if len(resp.RegistryChildren) == 0 {
		t.Fatalf("registry empty after launch; raw=%q", resp.RegistryRaw)
	}
	var match *RegistryChild
	for i := range resp.RegistryChildren {
		c := &resp.RegistryChildren[i]
		if c.SessionID == resp.LaunchSession.ID {
			match = c
			break
		}
	}
	if match == nil {
		t.Fatalf("no registry entry for session %s: %+v", resp.LaunchSession.ID, resp.RegistryChildren)
	}
	if match.Kind != "headless-agent" {
		t.Fatalf("kind = %q", match.Kind)
	}
	if match.AgentID != "grok" {
		t.Fatalf("agent_id = %q", match.AgentID)
	}
	if match.Port != resp.LaunchSession.Port {
		t.Fatalf("registry port %d != session port %d", match.Port, resp.LaunchSession.Port)
	}
	if match.PID <= 0 {
		t.Fatal("registry pid missing")
	}
	if !resp.PortListening {
		t.Fatalf("session port %d not listening", resp.LaunchSession.Port)
	}
}
```
