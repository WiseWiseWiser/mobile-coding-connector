## Expected

1. `IsBlockingMenu` is false.
2. `WritableReady` is true.
3. `WritableState` is `idle`.
4. `WritableReason` does not mention update available.

## Errors

- Still `loading` with reason `codex update available` (root cause of post-Skip hang).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.IsBlockingMenu {
		t.Fatalf("IsBlockingMenu=true for banner-alone (stripped model loading)")
	}
	if !resp.WritableReady || resp.WritableState != "idle" {
		t.Fatalf("want ready idle with residual banner only, got ready=%v state=%q reason=%q",
			resp.WritableReady, resp.WritableState, resp.WritableReason)
	}
	if strings.Contains(strings.ToLower(resp.WritableReason), "update") {
		t.Fatalf("idle banner reason should not mention update: %q", resp.WritableReason)
	}
}
```
