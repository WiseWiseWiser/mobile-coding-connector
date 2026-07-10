## Expected

1. `CanRun` is `false`.

## Errors

- Enabling Backup Now with no server endpoint.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanRun {
		t.Fatal("CanRun = true, want false without endpoint")
	}
}
```
