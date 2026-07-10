## Expected

1. `EnableActive` is false.
2. `DisableActive` is true.

## Errors

- Enable clickable while already on, or Disable greyed when on.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.EnableActive {
		t.Fatal("EnableActive = true, want false when task on")
	}
	if !resp.DisableActive {
		t.Fatal("DisableActive = false, want true when task on")
	}
}
```
