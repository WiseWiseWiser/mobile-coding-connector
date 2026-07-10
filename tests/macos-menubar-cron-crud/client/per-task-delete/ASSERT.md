## Expected

1. `HasPerTaskDelete` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Missing Delete… action in nested task menu.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasPerTaskDelete {
		t.Fatalf("missing per-task Delete… (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
