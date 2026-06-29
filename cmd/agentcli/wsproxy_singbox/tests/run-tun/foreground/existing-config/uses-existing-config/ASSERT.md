## Expected

- `Run` succeeds.
- `FetchVMessCalled` is false.
- `RunSingBoxConfig` equals `existing-singbox.json` path.

## Side Effects

- User config file unchanged.

## Errors

- None.

## Exit Code

- Success.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("run-tun error: %v", resp.RunErr)
	}
	if resp.FetchVMessCalled {
		t.Fatal("FetchVMess must be skipped with --config")
	}
	if !resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must be called")
	}
	if !strings.HasSuffix(resp.RunSingBoxConfig, "existing-singbox.json") {
		t.Fatalf("config path = %q, want existing-singbox.json", resp.RunSingBoxConfig)
	}
}
```