## Expected

1. `IsBlockingMenu` is true.
2. `MenuSelection` is `UPDATE_NOW`.
3. `WritableReady` is false.
4. `WritableState` is `loading` (or any non-`idle`).
5. `WritableReason` mentions update (substring `update`, case-insensitive).

## Errors

- Classified as residual banner / idle (would inject `/status` into modal).
- Selection reported as `SKIP` (Enter would be unsafe without re-check).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.IsBlockingMenu {
		t.Fatalf("IsBlockingMenu=false for default update modal (fixture %s)", req.FixtureFile)
	}
	if resp.MenuSelection != "UPDATE_NOW" {
		t.Fatalf("MenuSelection=%q, want UPDATE_NOW", resp.MenuSelection)
	}
	if resp.WritableReady {
		t.Fatalf("writable ready=true on blocking menu (state=%q reason=%q)",
			resp.WritableState, resp.WritableReason)
	}
	if resp.WritableState == "idle" {
		t.Fatalf("writable state=idle on blocking menu (reason=%q)", resp.WritableReason)
	}
	if !strings.Contains(strings.ToLower(resp.WritableReason), "update") {
		t.Fatalf("writable reason=%q, want mention of update", resp.WritableReason)
	}
}
```
