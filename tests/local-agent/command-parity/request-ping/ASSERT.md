## Expected

1. Exit code 0.
2. Stdout trimmed equals `pong`.

## Side Effects

None.

## Errors

- Wrong body or HTTP error.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if strings.TrimSpace(resp.Stdout) != "pong" {
		t.Fatalf("stdout = %q, want pong", resp.Stdout)
	}
}
```