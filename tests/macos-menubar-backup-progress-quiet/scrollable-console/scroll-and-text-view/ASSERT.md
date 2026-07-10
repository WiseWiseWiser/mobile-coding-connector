## Expected

1. `HasScrollableConsole` is true (`NSScrollView` + `NSTextView` + `documentView`).

## Side Effects

- None (read-only source inspection).

## Errors

- Replacing the console with a non-scrollable control or removing documentView wiring.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasScrollableConsole {
		t.Fatalf("expected NSScrollView + NSTextView documentView (source: %s)", resp.ProgressWindowSource)
	}
}
```
