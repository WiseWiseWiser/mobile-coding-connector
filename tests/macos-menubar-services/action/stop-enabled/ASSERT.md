## Expected

1. `CanStop` is `true`.

## Errors

- Stop stays disabled despite pid>0.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.CanStop {
		t.Fatal("CanStop = false, want true when pid>0")
	}
}
```