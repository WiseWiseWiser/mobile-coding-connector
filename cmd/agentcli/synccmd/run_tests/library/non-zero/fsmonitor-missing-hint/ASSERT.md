## Expected

1. Harness err nil.
2. `RunPairErr` mentions `unison-fsmonitor`, `brew`, and `both sides` (or `local`/`remote`).
3. `Result.ExitCode` is 3; `Result.Message` mentions fsmonitor.
4. Exec called; state exitCode 3.

## Side Effects

- State message reflects missing fsmonitor.

## Errors

- Non-nil RunPair error with install hints (not bare `unison exit code 3` only).

## Exit Code

- Non-nil.

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
	if resp.RunPairErr == "" {
		t.Fatal("expected RunPair error when fsmonitor helper missing")
	}
	low := strings.ToLower(resp.RunPairErr)
	if !strings.Contains(low, "unison-fsmonitor") {
		t.Fatalf("error should mention unison-fsmonitor; got %q", resp.RunPairErr)
	}
	if !strings.Contains(low, "brew") {
		t.Fatalf("error should mention brew install hint; got %q", resp.RunPairErr)
	}
	if !strings.Contains(low, "both sides") && !(strings.Contains(low, "local") && strings.Contains(low, "remote")) {
		t.Fatalf("error should mention both sides / local+remote; got %q", resp.RunPairErr)
	}
	if strings.TrimSpace(resp.RunPairErr) == "unison exit code 3" {
		t.Fatalf("error must be richer than bare exit code; got %q", resp.RunPairErr)
	}
	if resp.Result.ExitCode != 3 {
		t.Fatalf("Result.ExitCode: got %d want 3", resp.Result.ExitCode)
	}
	if !strings.Contains(strings.ToLower(resp.Result.Message), "fsmonitor") {
		t.Fatalf("Result.Message should mention fsmonitor; got %q", resp.Result.Message)
	}
	if !resp.ExecCalled {
		t.Fatal("expected Exec to be called")
	}
	if !resp.StateExists || resp.StateExit == nil || *resp.StateExit != 3 {
		t.Fatalf("state exitCode want 3; json=%s", resp.StateJSON)
	}
}
```
