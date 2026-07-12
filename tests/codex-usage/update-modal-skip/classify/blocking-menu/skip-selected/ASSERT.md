## Expected

1. `IsBlockingMenu` is true (menu still present).
2. `MenuSelection` is `SKIP`.
3. `WritableReady` is false (still blocking until Enter dismisses menu).
4. `WritableState` is not `idle`.

## Errors

- Selection `UPDATE_NOW` (must not Enter).
- Not classified as blocking menu.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.IsBlockingMenu {
		t.Fatalf("IsBlockingMenu=false for skip-selected menu")
	}
	if resp.MenuSelection != "SKIP" {
		t.Fatalf("MenuSelection=%q, want SKIP (verify-before-Enter)", resp.MenuSelection)
	}
	if resp.WritableReady || resp.WritableState == "idle" {
		t.Fatalf("skip-selected menu must still be non-idle (ready=%v state=%q reason=%q)",
			resp.WritableReady, resp.WritableState, resp.WritableReason)
	}
}
```
