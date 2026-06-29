## Expected

- `Run` succeeds.
- `StartDetachedSudo = false`.
- PID printed on stdout.

## Side Effects

- Background process started without sudo wrapper.

## Errors

- None.

## Exit Code

- Success.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("run-tun detach error: %v", resp.RunErr)
	}
	if !resp.StartDetachedCalled {
		t.Fatal("StartDetached must be called")
	}
	if resp.StartDetachedSudo {
		t.Fatal("root detach must not use sudo")
	}
	if !strings.Contains(resp.Stdout, "4242") {
		t.Fatalf("stdout must contain PID; got %q", resp.Stdout)
	}
}
```