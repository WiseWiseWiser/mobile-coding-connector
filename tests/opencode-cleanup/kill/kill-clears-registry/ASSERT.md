## Expected

1. `CleanupErr` is nil.
2. `RegistryEmpty` true.
3. `PortListening` false.
4. Fake opencode process not alive.

## Errors

- Non-empty registry after cleanup fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.CleanupErr != nil {
		t.Fatalf("CleanupOpencodeServe error: %v", resp.CleanupErr)
	}
	if !resp.RegistryEmpty {
		t.Fatalf("registry not empty after cleanup: %q", resp.RegistryRaw)
	}
	if resp.PortListening {
		t.Fatalf("port %d still listening", resp.FakeOpenCodePort)
	}
	if resp.ProcessAlive {
		t.Fatalf("fake opencode pid %d still alive", resp.FakeOpenCodePID)
	}
}
```
