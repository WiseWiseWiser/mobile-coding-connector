## Expected

- `RunErr` mentions `sing-box not installed`.
- `BrewInstallCalled` is false.
- `RunSingBoxCalled` is false.

## Side Effects

- None.

## Errors

- Install hint references Homebrew.

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
		t.Fatal("expected error when sing-box missing on non-TTY")
	}
	msg := strings.ToLower(resp.RunErr.Error())
	if !strings.Contains(msg, "sing-box not installed") && !strings.Contains(msg, "not installed") {
		t.Fatalf("error = %q, want not installed message", resp.RunErr)
	}
	if resp.BrewInstallCalled {
		t.Fatal("brew must not run on non-TTY")
	}
	if resp.RunSingBoxCalled {
		t.Fatal("RunSingBox must not run without binary")
	}
}
```