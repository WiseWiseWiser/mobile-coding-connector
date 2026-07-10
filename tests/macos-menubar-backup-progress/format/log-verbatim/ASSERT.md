## Expected

1. `Line` is exactly `dry-run: machine backup plan` (no `[log]` prefix).

## Errors

- Wrapping with `[log] ` or stripping the message.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "dry-run: machine backup plan"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
