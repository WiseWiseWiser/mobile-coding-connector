## Expected

1. `BuildOK` is true.
2. `Method` is `POST`.
3. `Path` is `/api/cron-tasks` (or URL ends with that path).
4. `BodyHasName` and `BodyHasCommand` are true.
5. `BodyCronExpr` is `0 1 * * *` (UTC, as provided).
6. `AuthHeader` is `Bearer secret-token` when token set.

## Errors

- Wrong method (PUT/GET); missing body fields; path under `/run` or query-id only.

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
	if resp.Method != "POST" {
		t.Fatalf("method = %q, want POST", resp.Method)
	}
	pathOK := resp.Path == "/api/cron-tasks" ||
		strings.HasSuffix(strings.TrimRight(resp.URL, "/"), "/api/cron-tasks")
	if !pathOK {
		t.Fatalf("path/url = %q %q, want /api/cron-tasks", resp.Path, resp.URL)
	}
	if !resp.BodyHasName {
		t.Fatalf("body missing name: %q", resp.Body)
	}
	if !resp.BodyHasCommand {
		t.Fatalf("body missing command: %q", resp.Body)
	}
	if resp.BodyCronExpr != "0 1 * * *" {
		t.Fatalf("body cronExpr = %q, want UTC 0 1 * * *", resp.BodyCronExpr)
	}
	if resp.AuthHeader != "Bearer secret-token" {
		t.Fatalf("auth = %q", resp.AuthHeader)
	}
}
```
