## Expected

1. `CanRun` is `true`.

## Errors

- Run Now enabled/disabled incorrectly for status `idle`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanRun != true {
		t.Fatalf("CanRun = %v, want %v for status %q", resp.CanRun, true, req.Status)
	}
}
```
