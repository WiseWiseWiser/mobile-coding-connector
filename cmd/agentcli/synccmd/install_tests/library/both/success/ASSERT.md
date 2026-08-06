## Expected

1. Harness err nil.
2. `InstallErr` empty.
3. Both LocalEnsure and RemoteEnsure called.
4. LocalOK and RemoteOK true; versions `2.54.0`.
5. `synccmd.PreferredUnisonVersion` equals `"2.54.0"`.

## Side Effects

- Both ensure hooks invoked.

## Errors

- None.

## Exit Code

- Nil library error.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/synccmd"
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
		t.Fatal("expected LocalEnsure to be called for scope both")
	}
	if !resp.RemoteEnsureCalled {
		t.Fatal("expected RemoteEnsure to be called for scope both")
	}
	if !resp.Report.LocalOK || !resp.Report.RemoteOK {
		t.Fatalf("want both OK true; report=%+v", resp.Report)
	}
	if resp.Report.LocalVersion != "2.54.0" {
		t.Fatalf("LocalVersion: got %q want %q", resp.Report.LocalVersion, "2.54.0")
	}
	if resp.Report.RemoteVersion != "2.54.0" {
		t.Fatalf("RemoteVersion: got %q want %q", resp.Report.RemoteVersion, "2.54.0")
	}
	if synccmd.PreferredUnisonVersion != wantPreferred {
		t.Fatalf("PreferredUnisonVersion: got %q want %q", synccmd.PreferredUnisonVersion, wantPreferred)
	}
}
```
