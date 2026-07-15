---
label: slow
explanation: real opencode serve when installed; registry poll after launch/stop
---

## Expected

1. `LaunchErr` is nil.
2. At launch: `LaunchRegistryChildren` has entry for session with matching `port` and `pid` > 0.
3. After stop: `RegistryEmpty` true.
4. `PortListening` false on session port.

## Errors

- Missing registry entry at launch or open port after stop fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.LaunchErr != nil {
		t.Fatalf("launch error: %v", resp.LaunchErr)
	}
	if resp.LaunchSession == nil {
		t.Fatal("expected session")
	}
	if len(resp.LaunchRegistryChildren) == 0 {
		t.Fatal("registry empty at launch")
	}
	found := false
	for _, c := range resp.LaunchRegistryChildren {
		if c.SessionID == resp.LaunchSession.ID {
			found = true
			if c.Port != resp.LaunchSession.Port {
				t.Fatalf("registry port %d != session port %d", c.Port, resp.LaunchSession.Port)
			}
			if c.PID <= 0 {
				t.Fatal("registry pid missing at launch")
			}
			break
		}
	}
	if !found {
		t.Fatalf("no registry entry for session %s at launch: %+v", resp.LaunchSession.ID, resp.LaunchRegistryChildren)
	}
	if resp.RegistryErr != nil {
		t.Fatalf("registry read error: %v", resp.RegistryErr)
	}
	if !resp.RegistryEmpty {
		t.Fatalf("registry not empty after stop: %+v", resp.RegistryChildren)
	}
	if resp.PortListening {
		t.Fatalf("port %d still listening after stop", resp.LaunchSession.Port)
	}
}
```
