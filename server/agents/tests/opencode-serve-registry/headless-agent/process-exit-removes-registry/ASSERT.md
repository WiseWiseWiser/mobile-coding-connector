## Expected

1. Launch succeeded.
2. After external kill: `RegistryEmpty` true within wait window.
3. `PortListening` false.

## Errors

- Stale registry entry after child death fails the test.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.LaunchErr != nil {
		t.Fatalf("launch error: %v", resp.LaunchErr)
	}
	if !resp.RegistryEmpty {
		t.Fatalf("registry still has entries after child exit: %+v", resp.RegistryChildren)
	}
	if resp.PortListening {
		t.Fatalf("port %d still listening after child exit", resp.SessionPort)
	}
}
```
