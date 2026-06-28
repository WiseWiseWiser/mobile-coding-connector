## Expected

- `RunErr` mentions sudo/TTY/root requirement.
- `RunSingBoxCalled` is false.

## Side Effects

- No sing-box process started.

## Errors

- Non-interactive environment cannot satisfy sudo password prompt.

## Exit Code

- Failure.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr == nil {
		t.Fatal("expected error when non-TTY needs sudo")
	}
	if resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must not be called without TTY for sudo")
	}
	msg := strings.ToLower(resp.RunErr.Error())
	if !strings.Contains(msg, "tty") && !strings.Contains(msg, "sudo") && !strings.Contains(msg, "root") {
		t.Fatalf("error = %q, want TTY/sudo/root hint", resp.RunErr)
	}
}
```