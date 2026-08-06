## Expected

1. Success.
2. Pair absent from store.
3. `ProfileExists` true (file retained).

## Side Effects

- Store without pair; profile file kept.

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
		t.Fatalf("rm error: %s", resp.RunErr)
	}
	if p := pairByName(resp, "mad-max"); p != nil {
		t.Fatalf("pair still present: %+v", p)
	}
	if !resp.ProfileExists {
		t.Fatalf("profile should remain with --no-purge-profile; path %s", resp.ProfilePath)
	}
}
```
