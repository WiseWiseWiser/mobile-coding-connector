## Expected

1. Create succeeds.
2. Listed task `Timeout` is `1h` (or a string equal to one hour, e.g. `60m` / `3600s` accepted only if product normalizes to `1h` — **prefer exact `1h`** per requirement).

## Side Effects

- Default applied at create time and visible on list/status.

## Errors

- Empty timeout, zero, or unlimited after create.

## Exit Code

0 from `Run`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("create failed: %s", resp.ActionError)
	}
	task, ok := findTaskByName(resp.Tasks, "default-timeout")
	if !ok {
		t.Fatalf("task missing: %+v", resp.Tasks)
	}
	if task.Timeout != "1h" {
		t.Fatalf("timeout=%q, want 1h", task.Timeout)
	}
}
```
