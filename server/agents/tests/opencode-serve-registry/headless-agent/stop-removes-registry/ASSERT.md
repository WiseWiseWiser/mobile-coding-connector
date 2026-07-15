## Expected

1. Launch succeeded (`LaunchErr` nil).
2. After stop: `RegistryEmpty` true (no children).
3. `PortListening` false on session port.

## Errors

- Registry entry survives stop or port still open fails the test.

```go
import "testing"

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
	if !resp.RegistryEmpty {
		t.Fatalf("registry not empty after stop: %+v raw=%q", resp.RegistryChildren, resp.RegistryRaw)
	}
	if resp.PortListening {
		t.Fatalf("port %d still listening after stop", resp.SessionPort)
	}
}
```
