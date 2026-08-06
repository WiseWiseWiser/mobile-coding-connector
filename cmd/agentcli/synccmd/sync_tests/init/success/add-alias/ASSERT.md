## Expected

1. `RunErr` empty.
2. Pair `demo` exists in store with backend `unison`.
3. Profile `remote-agent-demo.prf` exists.

## Side Effects

- Store and profile written for `demo`.

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
	if resp.RunErr != "" {
		t.Fatalf("RunCLI error: %s", resp.RunErr)
	}
	p := pairByName(resp, "demo")
	if p == nil {
		t.Fatalf("pair demo missing after add; json=%s", resp.PairsJSON)
	}
	if p.Backend != "unison" {
		t.Fatalf("backend: got %q want unison", p.Backend)
	}
	if !resp.ProfileExists {
		t.Fatalf("profile missing for add alias at %s", resp.ProfilePath)
	}
}
```
