## Expected

1. `BuildOK` is true.
2. `Method` is `GET`.
3. `URL` is `https://agent.example.com/api/cron-tasks` (trailing slash normalized).
4. `AuthHeader` is `Bearer secret-token`.
5. `HasAuth` is true.

## Errors

- Missing Authorization; keep-alive/loopback URL; wrong path.

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
	if resp.Method != "GET" {
		t.Fatalf("method = %q", resp.Method)
	}
	if resp.URL != "https://agent.example.com/api/cron-tasks" {
		t.Fatalf("url = %q", resp.URL)
	}
	if resp.AuthHeader != "Bearer secret-token" {
		t.Fatalf("auth = %q", resp.AuthHeader)
	}
	if !resp.HasAuth {
		t.Fatal("HasAuth = false")
	}
	if strings.Contains(resp.URL, "23312") || strings.Contains(resp.URL, "127.0.0.1") {
		t.Fatalf("must not target keep-alive: %q", resp.URL)
	}
}
```
