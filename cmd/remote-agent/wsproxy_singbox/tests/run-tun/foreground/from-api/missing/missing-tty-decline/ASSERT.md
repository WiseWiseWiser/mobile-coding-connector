## Expected

- `RunErr` is non-nil (install aborted).
- `ConfirmCalled` is true.
- `BrewInstallCalled` is false.
- `RunSingBoxCalled` is false.

## Side Effects

- None.

## Errors

- User declined install prompt.

## Exit Code

- Failure.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr == nil {
		t.Fatal("expected error when user declines install")
	}
	if !resp.ConfirmCalled {
		t.Fatal("confirm prompt must be shown on TTY")
	}
	if resp.BrewInstallCalled {
		t.Fatal("brew must not run after decline")
	}
	if resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must not run after decline")
	}
}
```