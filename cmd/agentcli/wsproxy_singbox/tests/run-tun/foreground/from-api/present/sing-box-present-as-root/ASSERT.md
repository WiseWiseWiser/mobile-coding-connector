## Expected

- `Run` succeeds.
- `RunSingBoxCalled` with `RunSingBoxSudo = false`.

## Side Effects

- None beyond sing-box invocation.

## Errors

- None.

## Exit Code

- Success.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("run-tun error: %v", resp.RunErr)
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must be called")
	}
	if resp.RunSingBoxSudo {
		t.Fatal("root must run sing-box without sudo")
	}
}
```