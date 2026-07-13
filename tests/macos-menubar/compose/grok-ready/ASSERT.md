## Expected

1. `DropdownLine` is exactly `Grok: 61%(Weekly), Reset July 17, 08:55, left 4d`.

## Errors

- Re-parsing raw next_reset, wrong separators, or missing `, left 4d`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok: 61%(Weekly), Reset July 17, 08:55, left 4d"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```
