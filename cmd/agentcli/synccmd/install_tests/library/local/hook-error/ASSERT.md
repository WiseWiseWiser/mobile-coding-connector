## Expected

1. Harness err nil (error is in InstallErr, not harness).
2. `InstallErr` non-empty and contains hook text (`brew` or `failed` or full message).
3. `LocalEnsureCalled` true; `RemoteEnsureCalled` false.
4. `Report.LocalOK` false.

## Side Effects

- LocalEnsure invoked (and failed).

## Errors

- Non-nil Install error surfacing hook failure.

## Exit Code

- Non-nil library error.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.InstallErr == "" {
		t.Fatal("expected Install error when LocalEnsure fails")
	}
	low := strings.ToLower(resp.InstallErr)
	if !strings.Contains(low, "brew") && !strings.Contains(low, "failed") &&
		!strings.Contains(resp.InstallErr, "unison formula missing") {
		t.Fatalf("Install error should surface hook failure; got %q", resp.InstallErr)
	}
	if !resp.LocalEnsureCalled {
		t.Fatal("expected LocalEnsure to be called")
	}
	if resp.RemoteEnsureCalled {
		t.Fatal("RemoteEnsure must not be called for scope local")
	}
	if resp.Report.LocalOK {
		t.Fatal("LocalOK should be false when LocalEnsure fails")
	}
}
```
