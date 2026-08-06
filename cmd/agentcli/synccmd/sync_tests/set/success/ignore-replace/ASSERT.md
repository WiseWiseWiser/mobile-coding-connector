## Expected

1. Success.
2. Pair.Ignore equals `["Name foo", "Path bar"]` (replace, not append to defaults).
3. Profile contains `ignore = Name foo` and `ignore = Path bar` (flexible spacing).
4. Profile does not still require default `Name .DS_Store` (may be absent after replace).

## Side Effects

- Store ignore + profile ignore lines replaced.

## Errors

- None.

## Exit Code

- Nil error.

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
	if resp.RunErr != "" {
		t.Fatalf("set error: %s", resp.RunErr)
	}
	p := pairByName(resp, "mad-max")
	if p == nil {
		t.Fatal("pair missing")
	}
	if len(p.Ignore) != 2 || p.Ignore[0] != "Name foo" || p.Ignore[1] != "Path bar" {
		t.Fatalf("ignore want [Name foo, Path bar]; got %v", p.Ignore)
	}
	if !resp.ProfileExists {
		t.Fatal("profile missing")
	}
	c := resp.ProfileContent
	if !strings.Contains(c, "Name foo") || !strings.Contains(c, "Path bar") {
		t.Fatalf("profile missing new ignore lines; content:\\n%s", c)
	}
}
```
