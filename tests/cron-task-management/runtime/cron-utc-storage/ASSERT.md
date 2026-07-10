## Expected

1. Create succeeds.
2. Listed task has `scheduleMode=cron` and `cronExpr=0 9 * * *` (unchanged).
3. `interval` empty/absent.

## Side Effects

- Definition persists UTC expr for server evaluation.

## Errors

- Expression rewritten or scheduleMode wrong.

## Exit Code

0 from `Run`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("create failed: %s body=%s", resp.ActionError, resp.Body)
	}
	task, ok := findTaskByName(resp.Tasks, "morning-utc")
	if !ok {
		t.Fatalf("task missing: %+v", resp.Tasks)
	}
	if task.ScheduleMode != "cron" {
		t.Fatalf("scheduleMode=%q, want cron", task.ScheduleMode)
	}
	if task.CronExpr != "0 9 * * *" {
		t.Fatalf("cronExpr=%q, want %q", task.CronExpr, "0 9 * * *")
	}
	if task.Interval != "" {
		t.Fatalf("interval should be empty for cron mode, got %q", task.Interval)
	}
}
```
