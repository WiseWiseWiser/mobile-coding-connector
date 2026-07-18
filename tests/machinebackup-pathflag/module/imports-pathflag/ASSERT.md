## Expected

1. `Run` completes without error and `Response.Err` is empty.
2. `ImportsPathflag` is true.

## Side Effects

- None.

## Errors

- Absent pathflag import fails (expected RED pre-implementer).

## Exit Code

- N/A

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Err != "" {
		t.Fatalf("package import inspect: %s", resp.Err)
	}
	if !resp.ImportsPathflag {
		t.Fatalf("package %s must import %s; detail=%q", machinebackupPkg, pathflagImport, resp.Detail)
	}
}
```
