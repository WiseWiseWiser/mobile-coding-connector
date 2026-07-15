## Expected

1. Launch succeeded.
2. `CleanupErr` is nil.
3. `RegistryEmpty` true after CleanupAll.
4. `PortListening` false on session port.

## Errors

- Surviving child or non-empty registry fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.LaunchErr != nil {
		t.Fatalf("launch error: %v", resp.LaunchErr)
	}
	if resp.CleanupErr != nil {
		t.Fatalf("CleanupAll error: %v", resp.CleanupErr)
	}
	if !resp.RegistryEmpty {
		t.Fatalf("registry not empty after CleanupAll: %+v", resp.RegistryChildren)
	}
	if resp.PortListening {
		t.Fatalf("port %d still listening after CleanupAll", resp.SessionPort)
	}
}
```
