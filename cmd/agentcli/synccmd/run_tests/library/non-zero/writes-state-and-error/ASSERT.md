## Expected

1. Harness err nil.
2. `RunPairErr` non-empty and mentions exit code (`3` or `exit`).
3. `Result.ExitCode` is 3.
4. Exec called.
5. State exists with exitCode 3 and non-empty lastRunAt.

## Side Effects

- `{StoreDir}/state/mad-max.json` with exitCode 3.

## Errors

- Non-nil RunPair error containing exit information.

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
		t.Fatal("expected RunPair error on non-zero Exec exit")
	}
	low := strings.ToLower(resp.RunPairErr)
	if !strings.Contains(resp.RunPairErr, "3") && !strings.Contains(low, "exit") {
		t.Fatalf("error should mention exit code 3 or exit; got %q", resp.RunPairErr)
	}
	if resp.Result.ExitCode != 3 {
		t.Fatalf("Result.ExitCode: got %d want 3", resp.Result.ExitCode)
	}
	if !resp.ExecCalled {
		t.Fatal("expected Exec to be called")
	}
	if !resp.StateExists {
		t.Fatalf("expected state file after non-zero exit at %s", resp.StatePath)
	}
	if resp.StateExit == nil {
		t.Fatalf("state exitCode nil; json=%s", resp.StateJSON)
	}
	if *resp.StateExit != 3 {
		t.Fatalf("state exitCode: got %d want 3; json=%s", *resp.StateExit, resp.StateJSON)
	}
	if resp.StateLastRun == "" {
		t.Fatalf("state lastRunAt empty; json=%s", resp.StateJSON)
	}
}
```
