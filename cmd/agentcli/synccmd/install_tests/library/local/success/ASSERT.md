## Expected

1. Harness err nil.
2. `InstallErr` empty.
3. `LocalEnsureCalled` true; `RemoteEnsureCalled` false.
4. `Report.LocalOK` true; `Report.LocalVersion` is `2.54.0`.
5. `Report.RemoteOK` false (remote not requested).

## Side Effects

- LocalEnsure invoked once (via harness capture).

## Errors

- None.

## Exit Code

- Nil library error.

```go
import (
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
	if resp.InstallErr != "" {
		t.Fatalf("Install error: %s", resp.InstallErr)
	}
	if !resp.LocalEnsureCalled {
		t.Fatal("expected LocalEnsure to be called for scope local")
	}
	if resp.RemoteEnsureCalled {
		t.Fatal("RemoteEnsure must not be called for scope local")
	}
	if !resp.Report.LocalOK {
		t.Fatalf("LocalOK: got false want true; report=%+v", resp.Report)
	}
	if resp.Report.LocalVersion != "2.54.0" {
		t.Fatalf("LocalVersion: got %q want %q", resp.Report.LocalVersion, "2.54.0")
	}
	if resp.Report.RemoteOK {
		t.Fatal("RemoteOK should be false when scope is local only")
	}
}
```
