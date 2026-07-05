## Expected

1. `DropdownLine` is exactly `Grok: Weekly Limit: 6% (Reset July 9, 16:55 PT)`.

## Errors

- Wrong prefix, punctuation, or reset formatting.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok: Weekly Limit: 6% (Reset July 9, 16:55 PT)"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```