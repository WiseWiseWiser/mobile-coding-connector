## Expected

1. No Run error.
2. `Excluded` is false.
3. `Reason` is empty.

## Side Effects

- None.

## Errors

- Unexpected skip fails.

## Exit Code

- N/A

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Excluded {
		t.Fatalf("IsExcluded(%q)=true (reason=%q), want false", req.RelPath, resp.Reason)
	}
	if resp.Reason != "" {
		t.Fatalf("ReasonFor(%q)=%q, want empty", req.RelPath, resp.Reason)
	}
}
```
