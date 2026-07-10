## Expected

1. `Title` / `Line` is exactly `Backup: (no server)`.

## Errors

- Empty title, `Backup: `, or different placeholder wording.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Backup: (no server)"
	if resp.Title != want && resp.Line != want {
		t.Fatalf("Title/Line = %q / %q, want %q", resp.Title, resp.Line, want)
	}
}
```
