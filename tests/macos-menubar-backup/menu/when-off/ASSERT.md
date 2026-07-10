## Expected

1. `EnableActive` is true.
2. `DisableActive` is false.

## Errors

- Disable clickable while already off, or Enable greyed when off.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.EnableActive {
		t.Fatal("EnableActive = false, want true when task off")
	}
	if resp.DisableActive {
		t.Fatal("DisableActive = true, want false when task off")
	}
}
```
