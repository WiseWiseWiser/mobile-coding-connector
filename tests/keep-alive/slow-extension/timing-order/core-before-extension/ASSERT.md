## Expected

1. `Response.CoreReadyMs` >= 0 and `Response.ExtensionStartMs` >= 0 (markers present).
2. `Response.CoreReadyMs < Response.ExtensionStartMs`.
3. `Response.ServerReady` is true.

## Side Effects

- None beyond standard harness.

## Errors

- Missing bootstrap markers after implementation should fail this leaf (RED until fixed).

## Exit Code

- `0` when ordering invariant holds.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ServerReady {
		t.Fatal("server not ready")
	}
	if resp.CoreReadyMs < 0 {
		t.Fatal("missing [bootstrap] phase=core_ready marker")
	}
	if resp.ExtensionStartMs < 0 {
		t.Fatal("missing [bootstrap] phase=extension_start marker")
	}
	if resp.CoreReadyMs >= resp.ExtensionStartMs {
		t.Fatalf("core_ready t_ms=%d must be < extension_start t_ms=%d", resp.CoreReadyMs, resp.ExtensionStartMs)
	}
}
```