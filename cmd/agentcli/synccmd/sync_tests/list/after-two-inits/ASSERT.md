## Expected

1. `PreErr` and `RunErr` empty.
2. Stdout contains `alpha` and `beta`.
3. Config has two pairs.

## Side Effects

- pairs.json has both entries.

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
		t.Fatalf("list error: %s", resp.RunErr)
	}
	if !strings.Contains(resp.Stdout, "alpha") || !strings.Contains(resp.Stdout, "beta") {
		t.Fatalf("list stdout must contain alpha and beta; got:\\n%s", resp.Stdout)
	}
	if resp.Config == nil || len(resp.Config.Pairs) != 2 {
		t.Fatalf("want 2 pairs in store, got %+v", resp.Config)
	}
}
```
