## Expected

1. `EmptyLabel` is exactly `No recent backups`.

## Errors

- Blank row, different wording, or reusing Terminals/Services empty labels.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "No recent backups"
	if resp.EmptyLabel != want {
		t.Fatalf("EmptyLabel = %q, want %q", resp.EmptyLabel, want)
	}
}
```
