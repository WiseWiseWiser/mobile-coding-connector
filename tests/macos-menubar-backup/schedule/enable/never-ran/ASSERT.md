## Expected

1. `ShouldRun` is `true` (run immediately on enable).

## Errors

- Deferring first run until next hour tick.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShouldRun {
		t.Fatal("ShouldRun = false, want true when never ran")
	}
}
```
