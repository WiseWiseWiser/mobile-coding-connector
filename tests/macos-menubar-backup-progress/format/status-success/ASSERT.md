## Expected

1. `Line` is exactly `Status: Success`.

## Errors

- `OK`, lowercase, or status-menu style `Status: On · …`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Status: Success"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
