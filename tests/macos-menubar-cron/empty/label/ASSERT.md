## Expected

1. `EmptyLabel` is exactly `No cron tasks configured`.

## Errors

- Missing or alternate placeholder (`No tasks`, `Empty`, etc.).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "No cron tasks configured"
	if resp.EmptyLabel != want {
		t.Fatalf("empty label = %q, want %q", resp.EmptyLabel, want)
	}
}
```
