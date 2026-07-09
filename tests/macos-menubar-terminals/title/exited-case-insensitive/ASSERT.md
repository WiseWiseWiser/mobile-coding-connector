## Expected

1. `Title` is exactly `demo [EXITED]` (case-insensitive/trimmed status still suffixes).

## Errors

- Treating only lowercase `exited` as exited, or echoing the raw status string into the title.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "demo [EXITED]"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
