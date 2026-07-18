## Expected

1. No Run error.
2. `Excluded` is true for `.ai-critic/service.log`.
3. `Reason` is non-empty (pathflag: log files).

## Side Effects

- None.

## Errors

- Included log (public API) fails — expected RED before pathflag wiring.

## Exit Code

- N/A

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !resp.Excluded {
		t.Fatalf("IsExcluded(%q)=false, want true (pathflag **/*.log via public API)", req.RelPath)
	}
	if resp.Reason == "" {
		t.Fatalf("ReasonFor(%q) empty, want log reason", req.RelPath)
	}
}
```
