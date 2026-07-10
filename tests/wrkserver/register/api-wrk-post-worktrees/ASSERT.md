## Expected

1. Status is **not** `404` (route is mounted).
2. Status is 4xx (validation for missing `project_path`) with non-empty `error`,
   **or** any non-404 handler response proving the POST leaf is wired.

## Errors

- 404 — `POST {base}/worktrees` not registered.
- Silent mux miss without JSON error body when validation expected.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == 404 {
		t.Fatalf("POST /api/wrk/worktrees returned 404 — Register did not mount leaf; body=%s", resp.Body)
	}
	// Empty body should be a validation failure, not success.
	if resp.StatusCode < 400 || resp.StatusCode >= 500 {
		t.Fatalf("status = %d, want 4xx validation (route hit); body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Error == "" {
		t.Fatalf("expected error JSON from CreateWorktree validation; body=%s", resp.Body)
	}
}
```
