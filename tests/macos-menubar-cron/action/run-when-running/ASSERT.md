## Expected

1. `CanRun` is `false`.

## Errors

- Run Now enabled/disabled incorrectly for status `running`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanRun != false {
		t.Fatalf("CanRun = %v, want %v for status %q", resp.CanRun, false, req.Status)
	}
}
```
