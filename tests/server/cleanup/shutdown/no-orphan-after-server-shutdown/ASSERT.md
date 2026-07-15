## Expected

1. Launch produced `SessionPort` > 0.
2. After SIGTERM shutdown: `PortListening` false.
3. `RegistryEmpty` true.

## Errors

- Orphan opencode serve on session port fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.SessionPort <= 0 {
		t.Fatal("session port not recorded")
	}
	if resp.PortListening {
		t.Fatalf("orphan listener on port %d after server shutdown", resp.SessionPort)
	}
	if !resp.RegistryEmpty {
		t.Fatalf("registry not empty after shutdown: %q", resp.RegistryRaw)
	}
}
```
