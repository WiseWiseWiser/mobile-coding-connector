## Expected

1. `Title` is exactly `nightly ○ Idle · cron 0 1 * * *`.

## Errors

- Wrong glyph, status word, schedule suffix, separators, or spacing.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := `nightly ○ Idle · cron 0 1 * * *`
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
