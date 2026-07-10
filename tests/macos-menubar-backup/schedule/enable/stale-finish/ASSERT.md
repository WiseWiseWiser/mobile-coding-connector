## Expected

1. `ShouldRun` is `true` (stale finish → run on enable).

## Errors

- Treating any prior finish as “enable only”.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShouldRun {
		t.Fatal("ShouldRun = false, want true when last finish > 1h ago")
	}
}
```
