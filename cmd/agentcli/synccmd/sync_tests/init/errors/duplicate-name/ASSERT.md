## Expected

1. `PreErr` empty; `RunErr` non-empty containing `already exists`.
2. Still exactly one pair named `mad-max` (not duplicated entries).

## Side Effects

- First init's store retained.

## Errors

- Substring: `already exists`.

## Exit Code

- Non-nil error on second init.

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
	if resp.PreErr != "" {
		t.Fatalf("pre-init failed: %s", resp.PreErr)
	}
	if resp.RunErr == "" {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(resp.RunErr, "already exists") {
		t.Fatalf("error should contain already exists; got %q", resp.RunErr)
	}
	if resp.Config == nil {
		t.Fatal("config nil after duplicate attempt")
	}
	n := 0
	for _, p := range resp.Config.Pairs {
		if p.Name == "mad-max" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want exactly 1 mad-max pair, got %d", n)
	}
}
```
