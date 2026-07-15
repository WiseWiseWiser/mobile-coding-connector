## Expected

1. `CustomErr` is nil.
2. Registry has entry with `kind=custom-agent` and matching `session_id`.
3. Registry `port` equals `CustomLaunch.Port`.
4. Session port is listening.

## Errors

- Missing custom-agent entry fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.CustomErr != nil {
		t.Fatalf("LaunchCustomAgent error: %v", resp.CustomErr)
	}
	if resp.CustomLaunch == nil {
		t.Fatal("expected custom launch result")
	}
	if resp.RegistryErr != nil {
		t.Fatalf("read registry error: %v", resp.RegistryErr)
	}
	var match *RegistryChild
	for i := range resp.RegistryChildren {
		c := &resp.RegistryChildren[i]
		if c.SessionID == resp.CustomLaunch.SessionID {
			match = c
			break
		}
	}
	if match == nil {
		t.Fatalf("no registry entry for custom session %s: %+v", resp.CustomLaunch.SessionID, resp.RegistryChildren)
	}
	if match.Kind != "custom-agent" {
		t.Fatalf("kind = %q, want custom-agent", match.Kind)
	}
	if match.Port != resp.CustomLaunch.Port {
		t.Fatalf("registry port %d != launch port %d", match.Port, resp.CustomLaunch.Port)
	}
	if match.PID <= 0 {
		t.Fatal("registry pid missing")
	}
	if !resp.PortListening {
		t.Fatalf("custom agent port %d not listening", resp.CustomLaunch.Port)
	}
}
```
