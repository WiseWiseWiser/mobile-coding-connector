## Expected

1. `PresentsWindow` is true (`NSWindow` plus an order-front family call).

## Side Effects

- None (read-only source inspection).

## Errors

- Dead open path with no window or no presentation call.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.PresentsWindow {
		t.Fatalf("expected NSWindow + order-front presentation (source: %s)", resp.ProgressWindowSource)
	}
}
```
