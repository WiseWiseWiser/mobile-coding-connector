## Expected

1. `RunErr` is empty.
2. Stderr contains `[custom] hello log`.
3. Stdout does **not** contain `  hello log` (default log formatter bypassed).
4. Other event types from mock without Print flags produce no stdout (section/progress ignored).

## Side Effects

None.

## Errors

- Default log formatter still writes to stdout when `Printer.Log` is set.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("streamcmd.Run error: %s", resp.RunErr)
	}
	if !strings.Contains(resp.Stderr, "[custom] hello log") {
		t.Fatalf("stderr missing custom log; got stderr=%q stdout=%q", resp.Stderr, resp.Stdout)
	}
	if strings.Contains(resp.Stdout, "  hello log") {
		t.Fatalf("stdout should not contain default log formatting when Printer.Log is set; stdout=%q", resp.Stdout)
	}
}
```
