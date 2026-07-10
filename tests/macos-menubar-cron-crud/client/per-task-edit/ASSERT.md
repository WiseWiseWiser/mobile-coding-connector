## Expected

1. `HasPerTaskEdit` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Missing Edit…; only top-level edit without per-task placement.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasPerTaskEdit {
		t.Fatalf("missing per-task Edit… (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
