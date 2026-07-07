## Expected

- `RunTun` succeeds.
- `EnsureSudoSetupCalled` is false.
- `RunSingBoxCalled` is true with sudo.

## Side Effects

- No auto-setup hook invocation.

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
	if resp.EnsureSudoSetupCalled {
		t.Fatal("EnsureSudoSetup must not run when --no-setup-sudo is set")
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must still run")
	}
	if !resp.RunSingBoxSudo {
		t.Fatal("non-root must still use sudo for sing-box")
	}
}
```