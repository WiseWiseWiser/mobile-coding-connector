## Expected

1. Success.
2. Pair.LocalHostname is `mac-mad-max`.

## Side Effects

- pairs.json updated.

## Errors

- None.

## Exit Code

- Nil error.

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
	if resp.PreErr != "" {
		t.Fatalf("pre-init failed: %s", resp.PreErr)
	}
	if resp.RunErr != "" {
		t.Fatalf("set error: %s", resp.RunErr)
	}
	p := pairByName(resp, "mad-max")
	if p == nil {
		t.Fatal("pair missing")
	}
	if p.LocalHostname != "mac-mad-max" {
		t.Fatalf("localHostname: got %q want mac-mad-max", p.LocalHostname)
	}
}
```
