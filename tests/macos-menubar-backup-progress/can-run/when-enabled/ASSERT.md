## Expected

1. `CanRun` is `true`.

## Errors

- Disabling Backup Now while the task is on (should stay available).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.CanRun {
		t.Fatal("CanRun = false, want true when enabled and ready")
	}
}
```
