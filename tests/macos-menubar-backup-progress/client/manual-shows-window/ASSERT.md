## Expected

1. `ManualShowsWindow` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Backup Now runs with no window open.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ManualShowsWindow {
		t.Fatalf("manual Backup Now does not open progress window (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
