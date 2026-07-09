## Expected

1. `AgentBinary` is exactly `local-agent`.

## Errors

- Returning `remote-agent` or a path-qualified binary for the pure helper.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AgentBinary != "local-agent" {
		t.Fatalf("agent binary = %q, want %q", resp.AgentBinary, "local-agent")
	}
}
```
