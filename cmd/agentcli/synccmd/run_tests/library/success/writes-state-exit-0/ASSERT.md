## Expected

1. Harness err nil.
2. `RunPairErr` empty.
3. `Result.ExitCode` is 0.
4. Exec was called; captured env has `UNISONLOCALHOSTNAME=remote-agent-mad-max`.
5. State file exists with exitCode 0 and non-empty lastRunAt.

## Side Effects

- `{StoreDir}/state/mad-max.json` created.

## Errors

- None.

## Exit Code

- Nil library error.

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
	if resp.RunPairErr != "" {
		t.Fatalf("RunPair error: %s", resp.RunPairErr)
	}
	if resp.Result.ExitCode != 0 {
		t.Fatalf("Result.ExitCode: got %d want 0", resp.Result.ExitCode)
	}
	if !resp.ExecCalled {
		t.Fatal("expected Exec to be called on success path")
	}
	if !envHas(resp.ExecEnv, "UNISONLOCALHOSTNAME", "remote-agent-mad-max") {
		// Also accept env from Result.Argv path if product only returns env via Build;
		// primary contract is child env passed to Exec.
		t.Fatalf("Exec env missing UNISONLOCALHOSTNAME=remote-agent-mad-max; env=%v", resp.ExecEnv)
	}
	if !resp.StateExists {
		t.Fatalf("expected state file at %s", resp.StatePath)
	}
	if resp.StateExit == nil {
		t.Fatalf("state missing exitCode; json=%s", resp.StateJSON)
	}
	if *resp.StateExit != 0 {
		t.Fatalf("state exitCode: got %d want 0; json=%s", *resp.StateExit, resp.StateJSON)
	}
	if resp.StateLastRun == "" {
		t.Fatalf("state lastRunAt empty; json=%s", resp.StateJSON)
	}
}
```
