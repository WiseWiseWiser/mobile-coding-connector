## Expected

- `RunTun` succeeds.
- `EnsureSudoSetupCalled` is true.
- `SudoSetupSkipped` is false (install path ran).
- `RunSingBoxCalled` is true with `RunSingBoxSudo=true`.

## Side Effects

- Auto-setup hook invoked with sing-box path before foreground run.

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
		t.Fatal("EnsureSudoSetup must be called by default when needSudo")
	}
	if resp.SudoSetupSkipped {
		t.Fatal("SudoSetupSkipped must be false when not yet installed")
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must be called after auto-setup")
	}
	if !resp.RunSingBoxSudo {
		t.Fatal("non-root must run sing-box via sudo")
	}
}
```