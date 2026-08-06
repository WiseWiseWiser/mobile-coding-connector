## Expected

1. `RunErr` empty.
2. Stdout does not invent a fake pair name `mad-max` (no pairs exist).
3. Optional: message containing `empty` or `no pair` is fine but not required.

## Side Effects

- No requirement to create pairs.json.

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
	if resp.RunErr != "" {
		t.Fatalf("list empty should succeed; got %s", resp.RunErr)
	}
	if strings.Contains(resp.Stdout, "mad-max") {
		t.Fatalf("empty list must not print mad-max; got:\\n%s", resp.Stdout)
	}
}
```
