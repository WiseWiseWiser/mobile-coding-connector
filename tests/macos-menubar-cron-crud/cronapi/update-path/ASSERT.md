## Expected

1. `BuildOK` is true.
2. `Method` is `PUT`.
3. Path/URL targets `/api/cron-tasks`.
4. `BodyHasID` is true (id `task-1` in JSON body).
5. `BodyHasName` is true.

## Errors

- POST instead of PUT; id only as query; missing id in body.

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
	if resp.Method != "PUT" {
		t.Fatalf("method = %q, want PUT", resp.Method)
	}
	pathOK := resp.Path == "/api/cron-tasks" ||
		strings.HasSuffix(strings.TrimRight(resp.URL, "/"), "/api/cron-tasks")
	if !pathOK {
		t.Fatalf("path/url = %q %q, want /api/cron-tasks", resp.Path, resp.URL)
	}
	if !resp.BodyHasID {
		t.Fatalf("body missing id: %q", resp.Body)
	}
	if !strings.Contains(resp.Body, "task-1") {
		t.Fatalf("body missing task-1: %q", resp.Body)
	}
	if !resp.BodyHasName {
		t.Fatalf("body missing name: %q", resp.Body)
	}
}
```
