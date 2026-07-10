## Expected

1. `Line` is exactly `[section] Collecting files`.

## Errors

- Missing brackets, wrong type tag, or Title: prefix from CLI streamcmd.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "[section] Collecting files"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
