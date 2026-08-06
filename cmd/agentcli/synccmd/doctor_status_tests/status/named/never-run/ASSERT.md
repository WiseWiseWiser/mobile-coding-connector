## Expected

1. Harness err nil.
2. `StatusErr` empty.
3. Exactly one item (or at least mad-max present).
4. Item Name `mad-max`; Local/Remote match seed paths.
5. `LastRun` contains `never` (case-insensitive).
6. `ServeOK` true.

## Side Effects

- None.

## Errors

- None.

## Exit Code

- Nil library error.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.StatusErr != "" {
		t.Fatalf("Status error: %s", resp.StatusErr)
	}
	it := statusNamed(resp.Status, "mad-max")
	if it == nil {
		t.Fatalf("missing status item mad-max; items=%+v", resp.Status.Items)
	}
	if it.Local != req.LocalPath {
		t.Fatalf("local: got %q want %q", it.Local, req.LocalPath)
	}
	if it.Remote != req.RemotePath {
		t.Fatalf("remote: got %q want %q", it.Remote, req.RemotePath)
	}
	if !strings.Contains(strings.ToLower(it.LastRun), "never") {
		t.Fatalf("LastRun want never; got %q", it.LastRun)
	}
	if !it.ServeOK {
		t.Fatal("ServeOK want true")
	}
}
```
