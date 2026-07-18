## Expected

1. No Run error.
2. `Excluded` is true.
3. `Reason` non-empty.

## Side Effects

- None.

## Errors

- Included path fails.

## Exit Code

- N/A

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !resp.Excluded {
		t.Fatalf("IsExcluded(%q)=false, want true", req.RelPath)
	}
	if resp.Reason == "" {
		t.Fatalf("ReasonFor(%q) empty", req.RelPath)
	}
}
```
