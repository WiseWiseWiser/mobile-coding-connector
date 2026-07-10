## Expected

1. `BuildOK` is true.
2. `Method` is `DELETE`.
3. `Path` starts with `/api/cron-tasks?` and includes id `task-1`.
4. `AuthHeader` is `Bearer secret-token`.

## Errors

- POST path like enable/disable; missing query id; GET.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.BuildOK {
		t.Fatalf("build failed: %s", resp.BuildErr)
	}
	if resp.Method != "DELETE" {
		t.Fatalf("method = %q, want DELETE", resp.Method)
	}
	path := resp.Path
	if path == "" {
		path = resp.URL
	}
	if !strings.Contains(path, "/api/cron-tasks?") {
		t.Fatalf("path/url = %q, want /api/cron-tasks?id=…", path)
	}
	if !strings.Contains(path, "task-1") {
		t.Fatalf("path missing id: %q", path)
	}
	if resp.AuthHeader != "Bearer secret-token" {
		t.Fatalf("auth = %q", resp.AuthHeader)
	}
}
```
