## Expected

- `Run` succeeds.
- `FetchVMessCalled` is true.
- `RunSingBoxCalled` is true with `RunSingBoxSudo = true`.
- Config path passed to `RunSingBox` is non-empty.

## Side Effects

- No brew install.

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
	if !resp.FetchVMessCalled {
		t.Fatal("FetchVMess must be called without --config")
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must be called")
	}
	if !resp.RunSingBoxSudo {
		t.Fatal("non-root must run sing-box via sudo")
	}
	if resp.RunSingBoxConfig == "" {
		t.Fatal("config path must be passed to RunSingBox")
	}
	if resp.BrewInstallCalled {
		t.Fatal("brew must not run when sing-box is on PATH")
	}
}
```