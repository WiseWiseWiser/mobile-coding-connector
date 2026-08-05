## Expected

1. Harness `err` is nil.
2. `RunnerErr` empty.
3. `Stdout` contains `EchoNeedle` (`hello`).
4. `AdhocPort` > 0.

## Side Effects

- AdhocServer closed after Run.

## Errors

None expected.

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
	if resp.RunnerErr != "" {
		t.Fatalf("CryptoSSHRunner.Run error: %s (stderr=%q)", resp.RunnerErr, resp.Stderr)
	}
	if resp.AdhocPort <= 0 {
		t.Fatalf("AdhocPort: got %d want > 0", resp.AdhocPort)
	}
	needle := req.EchoNeedle
	if needle == "" {
		needle = "hello"
	}
	if !strings.Contains(resp.Stdout, needle) {
		t.Fatalf("Stdout must contain %q; got %q (stderr=%q)", needle, resp.Stdout, resp.Stderr)
	}
}
```
