## Expected

1. No Run error.
2. `Excluded` is false for `.cache/x` when `.cache` is included.

## Side Effects

- None.

## Errors

- Still excluded fails.

## Exit Code

- N/A

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Excluded {
		t.Fatalf("IsExcluded(%q)=true with include .cache (reason=%q), want false", req.RelPath, resp.Reason)
	}
}
```
