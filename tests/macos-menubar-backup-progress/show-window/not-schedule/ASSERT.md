## Expected

1. `ShowWindow` is `true`.

## Errors

- Silent Backup Now (no progress UI for interactive runs).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShowWindow {
		t.Fatal("ShowWindow = false, want true for non-schedule runs")
	}
}
```
