## Expected

1. `Title` is exactly `scrape ⚠ Error · every 1m`.

## Errors

- Wrong glyph, status word, schedule suffix, separators, or spacing.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := `scrape ⚠ Error · every 1m`
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
