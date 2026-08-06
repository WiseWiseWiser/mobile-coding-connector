## Expected

1. Harness err nil.
2. `InstallErr` empty.
3. `RemoteEnsureCalled` true; `LocalEnsureCalled` false.
4. `RemoteEnsurePath` equals `req.RemoteTargetPath`.
5. `Report.RemoteOK` true; `Report.RemoteVersion` is `2.54.0`.
6. `Report.LocalOK` false (local not requested).

## Side Effects

- RemoteEnsure invoked with configured target path.

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
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.InstallErr != "" {
		t.Fatalf("Install error: %s", resp.InstallErr)
	}
	if !resp.RemoteEnsureCalled {
		t.Fatal("expected RemoteEnsure to be called for scope remote")
	}
	if resp.LocalEnsureCalled {
		t.Fatal("LocalEnsure must not be called for scope remote")
	}
	if req.RemoteTargetPath != "" && resp.RemoteEnsurePath != req.RemoteTargetPath {
		t.Fatalf("RemoteEnsure path: got %q want %q", resp.RemoteEnsurePath, req.RemoteTargetPath)
	}
	if !resp.Report.RemoteOK {
		t.Fatalf("RemoteOK: got false want true; report=%+v", resp.Report)
	}
	if resp.Report.RemoteVersion != "2.54.0" {
		t.Fatalf("RemoteVersion: got %q want %q", resp.Report.RemoteVersion, "2.54.0")
	}
	if resp.Report.LocalOK {
		t.Fatal("LocalOK should be false when scope is remote only")
	}
}
```
