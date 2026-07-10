## Expected

1. `Path` is exactly `/api/cron-tasks`.
2. `Method` is `GET`.

## Errors

- Wrong path (`/api/cron-tasks/`, query noise, or services path).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Path != "/api/cron-tasks" {
		t.Fatalf("path = %q, want /api/cron-tasks", resp.Path)
	}
	if resp.Method != "GET" {
		t.Fatalf("method = %q, want GET", resp.Method)
	}
}
```
