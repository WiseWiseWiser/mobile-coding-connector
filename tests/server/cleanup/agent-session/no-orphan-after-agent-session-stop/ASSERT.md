## Expected

1. `SessionPort` > 0 (launch succeeded).
2. `PortListening` false after stop + server teardown.
3. `RegistryEmpty` true.

## Errors

- Listener on session port after test completes fails the test.

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
		t.Fatalf("orphan listener still on session port %d", resp.SessionPort)
	}
	if !resp.RegistryEmpty {
		t.Fatalf("registry not empty: %q", resp.RegistryRaw)
	}
}
```
