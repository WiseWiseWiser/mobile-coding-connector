## Expected

- `RunErr` is non-nil mentioning sing-box not installed.
- `BrewInstallCalled` is false.
- `ConfirmCalled` is false.

## Side Effects

- None.

## Errors

- Fast fail per `--no-install`.

## Exit Code

- Failure.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr == nil {
		t.Fatal("expected error with --no-install and missing binary")
	}
	if resp.BrewInstallCalled {
		t.Fatal("brew must not run with --no-install")
	}
	if resp.ConfirmCalled {
		t.Fatal("confirm must not run with --no-install")
	}
	msg := strings.ToLower(resp.RunErr.Error())
	if !strings.Contains(msg, "not installed") && !strings.Contains(msg, "sing-box") {
		t.Fatalf("error = %q, want sing-box not installed", resp.RunErr)
	}
}
```