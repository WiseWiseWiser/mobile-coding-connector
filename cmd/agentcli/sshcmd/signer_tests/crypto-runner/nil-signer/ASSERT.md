## Expected

1. Harness `err` is nil.
2. `RunnerErr` is non-empty and contains `signer` (case-sensitive substring).
3. Command did not succeed (no requirement on stdout content).

## Side Effects

None (no network, no key files required).

## Errors

- Expected product error from CryptoSSHRunner.Run when Signer is nil.

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
	if resp.RunnerErr == "" {
		t.Fatal("CryptoSSHRunner with nil Signer must return an error")
	}
	if !strings.Contains(resp.RunnerErr, "signer") {
		t.Fatalf("RunnerErr must contain %q; got %q", "signer", resp.RunnerErr)
	}
}
```
