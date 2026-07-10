## Expected Output

CLI stdout includes the local expression and the stored UTC expression (order flexible).
Trailing newline on CLI content when matched with `assert.Output` full templates.

## Expected

1. CLI exit code 0.
2. Stdout (case-insensitive ok) contains both `0 9 * * *` (local input) and `0 1 * * *` (UTC stored).
3. Listed task `cronExpr` is `0 1 * * *` and `scheduleMode` is `cron`.

## Side Effects

- Task persisted with UTC cron expression.

## Errors

- Non-zero exit.
- Stored expr still local `0 9 * * *`.
- Stdout omits one of the expressions.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v\n%s", err, resp.Combined)
	}
	if resp.ActionError != "" {
		t.Fatalf("CLI error: %s\n%s", resp.ActionError, resp.Combined)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d\n%s", resp.ExitCode, resp.Combined)
	}

	// Flexible contains checks for both expressions (stdout must mention convert result).
	out := resp.Stdout
	if !strings.Contains(out, "0 9 * * *") {
		t.Fatalf("stdout missing local expr 0 9 * * *:\n%s", out)
	}
	if !strings.Contains(out, "0 1 * * *") {
		t.Fatalf("stdout missing stored UTC expr 0 1 * * *:\n%s", out)
	}
	assert.Output(t, out, `<contains>
0 1 * * *
</contains>`)

	task, ok := findTaskByName(resp.Tasks, "local-morning")
	if !ok {
		t.Fatalf("task not listed after CLI add: %+v", resp.Tasks)
	}
	if task.ScheduleMode != "cron" {
		t.Fatalf("scheduleMode=%q, want cron", task.ScheduleMode)
	}
	if task.CronExpr != "0 1 * * *" {
		t.Fatalf("stored cronExpr=%q, want 0 1 * * *", task.CronExpr)
	}
}
```
