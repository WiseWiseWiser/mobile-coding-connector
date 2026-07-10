## Expected

1. Create succeeds (`HTTPStatus` 2xx, no `ActionError`).
2. `Response.Tasks` contains a task named `echo-every-5m`.
3. That task has `scheduleMode=interval`, `interval=5m`, command containing `echo hello-cron`.
4. Task has non-empty `id` and `logPath`.

## Side Effects

- Definition persisted under config home `cron-tasks.json` (list reflects it).

## Errors

- Create fails or list missing the task.

## Exit Code

0 from `Run`.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v body=%s", err, resp.Body)
	}
	if resp.ActionError != "" {
		t.Fatalf("create failed: %s body=%s", resp.ActionError, resp.Body)
	}
	if resp.HTTPStatus < 200 || resp.HTTPStatus >= 300 {
		t.Fatalf("HTTP status %d body=%s", resp.HTTPStatus, resp.Body)
	}
	task, ok := findTaskByName(resp.Tasks, "echo-every-5m")
	if !ok {
		t.Fatalf("task echo-every-5m missing from list: %+v", resp.Tasks)
	}
	if task.ID == "" {
		t.Fatal("created task has empty id")
	}
	if task.ScheduleMode != "interval" {
		t.Fatalf("scheduleMode=%q, want interval", task.ScheduleMode)
	}
	if task.Interval != "5m" {
		t.Fatalf("interval=%q, want 5m", task.Interval)
	}
	if !strings.Contains(task.Command, "echo hello-cron") {
		t.Fatalf("command=%q, want to contain echo hello-cron", task.Command)
	}
	if task.LogPath == "" {
		t.Fatal("logPath empty")
	}
}
```
