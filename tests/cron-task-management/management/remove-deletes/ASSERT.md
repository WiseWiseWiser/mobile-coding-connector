## Expected

1. Delete succeeds (no `ActionError`, 2xx).
2. `Response.Tasks` does not contain id or name `to-remove`.

## Side Effects

- Definition removed from `cron-tasks.json`.

## Errors

- Task still listed after delete.

## Exit Code

0 from `Run`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("delete failed: %s", resp.ActionError)
	}
	if _, ok := findTaskByID(resp.Tasks, "to-remove"); ok {
		t.Fatal("to-remove still present by id after delete")
	}
	if _, ok := findTaskByName(resp.Tasks, "to-remove"); ok {
		t.Fatal("to-remove still present by name after delete")
	}
}
```
