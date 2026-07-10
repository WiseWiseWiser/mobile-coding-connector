## Expected

1. `ScrollOnlyOnFlush` is true (flush path scrolls; no per-line string+=/scroll pair).

## Side Effects

- None (read-only source inspection).

## Errors

- `scrollToEndOfDocument` on every append with `string +=` and no flush.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ScrollOnlyOnFlush {
		t.Fatalf("expected scrollToEnd only on flush path (source: %s)", resp.ProgressWindowSource)
	}
}
```
