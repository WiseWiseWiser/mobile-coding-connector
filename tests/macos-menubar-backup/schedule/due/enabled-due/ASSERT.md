## Expected

1. `ShouldRun` is `true`.

## Errors

- Skipping a due run while idle.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShouldRun {
		t.Fatal("ShouldRun = false, want true when enabled, due, not running")
	}
}
```
