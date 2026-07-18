## Expected

1. No Run error.
2. Primary path `.ai-critic/keep.log` is not excluded.
3. Secondary path `.ai-critic/service.log` is excluded (other logs still skip).

## Side Effects

- None.

## Errors

- keep.log excluded, or service.log included, fails.

## Exit Code

- N/A

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Excluded {
		t.Fatalf("IsExcluded(%q)=true with include override, want false", req.RelPath)
	}
	if !resp.SecondaryExcluded {
		t.Fatalf("IsExcluded(%q)=false, want true (other .log still skipped via catalog)", req.SecondaryRelPath)
	}
}
```
