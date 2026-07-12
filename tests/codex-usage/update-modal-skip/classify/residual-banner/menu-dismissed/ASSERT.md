## Expected

1. `IsBlockingMenu` is false (no menu options / no modal footer).
2. `MenuSelection` is empty (or not `UPDATE_NOW` / `SKIP`).
3. `WritableReason` must **not** be (or equal ignore-case) `codex update available`.
4. Because fixture still has `model: loading`, `WritableReady` may be false and
   `WritableState` may be `loading` — but the reason must mention **model**, not
   treat bare update banner as the update modal.

## Errors

- `IsBlockingMenu=true` (banner misclassified as modal).
- `WritableReason` contains `update available` (today’s bug: hang after Skip).

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
		t.Fatalf("IsBlockingMenu=true for residual banner (fixture %s) — menu options are gone",
			req.FixtureFile)
	}
	if resp.MenuSelection == "UPDATE_NOW" || resp.MenuSelection == "SKIP" {
		t.Fatalf("MenuSelection=%q on non-menu banner, want empty", resp.MenuSelection)
	}
	reason := strings.ToLower(strings.TrimSpace(resp.WritableReason))
	if reason == "codex update available" || strings.Contains(reason, "update available") {
		t.Fatalf("writable reason=%q: residual banner must not use update-available gate (today hangs waitForPrompt)",
			resp.WritableReason)
	}
	// Fixture still has model: loading — allow loading, but prefer model reason.
	if !resp.WritableReady {
		if resp.WritableState == "loading" && !strings.Contains(reason, "model") {
			t.Fatalf("non-ready residual banner should be model loading, got state=%q reason=%q",
				resp.WritableState, resp.WritableReason)
		}
	}
}
```
