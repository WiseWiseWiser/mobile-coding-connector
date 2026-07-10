## Expected

1. `StatusTitle` is exactly `Status: On · Error · 5m ago`.

## Errors

- Omitting Error label or raw error string in the title line.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Status: On · Error · 5m ago"
	if resp.StatusTitle != want {
		t.Fatalf("StatusTitle = %q, want %q", resp.StatusTitle, want)
	}
}
```
