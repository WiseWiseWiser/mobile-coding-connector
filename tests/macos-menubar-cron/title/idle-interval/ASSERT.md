## Expected

1. `Title` is exactly `backup ○ Idle · every 5m`.

## Errors

- Wrong glyph, status word, schedule suffix, separators, or spacing.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := `backup ○ Idle · every 5m`
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
