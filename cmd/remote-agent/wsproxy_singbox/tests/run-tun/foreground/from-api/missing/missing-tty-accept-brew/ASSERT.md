## Expected

- `Run` succeeds.
- `ConfirmCalled` is true.
- `BrewInstallCalled` is true.
- `RunSingBoxCalled` is true with sudo (non-root).

## Side Effects

- Homebrew install invoked via hook.

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
	if !resp.ConfirmCalled {
		t.Fatal("confirm prompt must be shown")
	}
	if !resp.BrewInstallCalled {
		t.Fatal("brew install must run after accept")
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must run after install")
	}
	if !resp.RunSingBoxSudo {
		t.Fatal("non-root must use sudo after install")
	}
}
```