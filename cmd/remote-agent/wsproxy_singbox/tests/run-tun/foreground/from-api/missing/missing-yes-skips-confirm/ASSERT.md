## Expected

- `Run` succeeds.
- `ConfirmCalled` is false (`--yes` bypasses prompt).
- `BrewInstallCalled` is true.
- `RunSingBoxCalled` is true.

## Side Effects

- Brew install without interactive confirm.

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
	if resp.ConfirmCalled {
		t.Fatal("--yes must skip confirm prompt")
	}
	if !resp.BrewInstallCalled {
		t.Fatal("brew must run with --yes when binary missing")
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must run after install")
	}
}
```