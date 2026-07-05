## Expected

1. `ConcurrentStarted` is `2` (both goroutines invoked refresh).
2. `FetchInvocationCount` is `1` (slow injectable fetch ran exactly once).

## Errors

- Overlapping refresh started multiple fetches (counter > 1).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ConcurrentStarted != 2 {
		t.Fatalf("ConcurrentStarted = %d, want 2", resp.ConcurrentStarted)
	}
	if resp.FetchInvocationCount != 1 {
		t.Fatalf("fetch invocations = %d, want 1 (overlap should be skipped)", resp.FetchInvocationCount)
	}
}
```