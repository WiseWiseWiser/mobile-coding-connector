## Expected

1. `ProgressEnqueueOnly` is true (callback uses append; session batches).

## Side Effects

- None (read-only source inspection).

## Errors

- Callback or session performs per-line `textView.string +=` / MainActor UI thrash
  with no enqueue/flush separation.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ProgressEnqueueOnly {
		t.Fatalf("expected onProgress to enqueue via session.append with batched flush (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
