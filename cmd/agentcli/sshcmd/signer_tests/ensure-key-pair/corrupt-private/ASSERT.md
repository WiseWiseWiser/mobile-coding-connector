## Expected

1. Harness `err` is nil (product error is in Response).
2. `EnsureErr` is non-empty (parse/load failure surfaced).
3. Prefer fail-closed: do not treat success + new Signer as pass.

## Side Effects

- Corrupt file remains under ConfigDir (test artifact).
- Implementer must not silently replace corrupt private with a new identity.

## Errors

- Expected: EnsureClientKeyPair returns a non-nil error (message may mention
  parse / private / key / ssh).

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
	if resp.EnsureErr == "" {
		t.Fatal("EnsureClientKeyPair on corrupt private must return an error; EnsureErr is empty (silent regen is forbidden)")
	}
}
```
