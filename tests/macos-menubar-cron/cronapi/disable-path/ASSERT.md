## Expected

1. `Method` is `POST`.
2. `Path` starts with `/api/cron-tasks/disable?`.
3. `Path` includes id `task-1` (as query value).

## Errors

- Missing id, wrong action segment, or GET method.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Method != "POST" {
		t.Fatalf("method = %q, want POST", resp.Method)
	}
	if !strings.HasPrefix(resp.Path, "/api/cron-tasks/disable?") {
		t.Fatalf("path = %q, want prefix %q", resp.Path, "/api/cron-tasks/disable?")
	}
	if !strings.Contains(resp.Path, "task-1") {
		t.Fatalf("path missing id: %q", resp.Path)
	}
}
```
