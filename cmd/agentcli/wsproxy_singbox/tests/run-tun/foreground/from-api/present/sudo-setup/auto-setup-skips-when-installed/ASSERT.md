## Expected

- `RunTun` succeeds.
- `EnsureSudoSetupCalled` is true.
- `SudoSetupSkipped` is true (already installed).
- `RunSingBoxCalled` is true.

## Side Effects

- No new sudoers install in hook (skipped path).

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
	if !resp.EnsureSudoSetupCalled {
		t.Fatal("EnsureSudoSetup should still be invoked to check install state")
	}
	if !resp.SudoSetupSkipped {
		t.Fatal("SudoSetupSkipped must be true when already installed")
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must still run after skip")
	}
}
```