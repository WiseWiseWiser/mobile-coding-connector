## Expected

1. No Run error.
2. `Excluded` is true.
3. `Reason` is non-empty (user excluded).

## Side Effects

- None.

## Errors

- Not excluded fails.

## Exit Code

- N/A

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !resp.Excluded {
		t.Fatalf("IsExcluded(%q)=false with exclude .docker, want true", req.RelPath)
	}
	if resp.Reason == "" {
		t.Fatalf("ReasonFor(%q) empty, want user excluded reason", req.RelPath)
	}
}
```
