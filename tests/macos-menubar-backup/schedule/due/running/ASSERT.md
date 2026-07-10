## Expected

1. `ShouldRun` is `false` (no overlap).

## Errors

- Parallel/overlapping backup runs for the same server.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ShouldRun {
		t.Fatal("ShouldRun = true, want false while already running")
	}
}
```
