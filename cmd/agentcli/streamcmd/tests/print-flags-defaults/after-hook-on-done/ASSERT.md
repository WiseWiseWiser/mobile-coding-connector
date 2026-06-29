## Expected

1. `RunErr` is empty.
2. `AfterCalled` is true.
3. `AfterDone["healthy"]` is `true`.
4. `AfterDone["binary_path"]` is `/tmp/x`.

## Side Effects

None.

## Errors

- `After` not invoked on successful stream.
- Done payload not passed to `After`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("streamcmd.Run error: %s", resp.RunErr)
	}
	if !resp.AfterCalled {
		t.Fatal("After hook was not called")
	}
	if healthy, ok := resp.AfterDone["healthy"].(bool); !ok || !healthy {
		t.Fatalf("AfterDone[healthy] = %v, want true", resp.AfterDone["healthy"])
	}
	if path, _ := resp.AfterDone["binary_path"].(string); path != "/tmp/x" {
		t.Fatalf("AfterDone[binary_path] = %q, want /tmp/x", path)
	}
}
```
