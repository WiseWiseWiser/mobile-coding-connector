## Expected

1. First `TimeLeft` is exactly `left 4d2h`.
2. Second `TimeLeftSecond` is exactly `left 4d` (2h later, no re-PTY).
3. `ResetAt` remains the seeded absolute instant (RFC3339, same value).
4. `ResetDisplay` remains non-empty (seeded UI token).

## Errors

- Same `time_left` on both Gets (not recomputed).
- Empty fields when hooks/implementation missing.
- Requiring a second fetch to advance countdown.

```go
import (
	"testing"
	"time"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 4d2h" {
		t.Fatalf("first time_left = %q, want %q", resp.TimeLeft, "left 4d2h")
	}
	if resp.TimeLeftSecond != "left 4d" {
		t.Fatalf("second time_left = %q, want %q", resp.TimeLeftSecond, "left 4d")
	}
	if resp.ResetAt == "" {
		t.Fatal("reset_at empty after Get recompute")
	}
	if _, parseErr := time.Parse(time.RFC3339, resp.ResetAt); parseErr != nil {
		t.Fatalf("reset_at not RFC3339: %q (%v)", resp.ResetAt, parseErr)
	}
	// Seeded absolute must be stable across Gets.
	if resp.ResetAt != req.ResetAtRFC3339 {
		// Allow equivalent RFC3339 forms if production normalizes; require same instant.
		want, werr := time.Parse(time.RFC3339, req.ResetAtRFC3339)
		got, gerr := time.Parse(time.RFC3339, resp.ResetAt)
		if werr != nil || gerr != nil || !want.Equal(got) {
			t.Fatalf("reset_at = %q, want same instant as seed %q", resp.ResetAt, req.ResetAtRFC3339)
		}
	}
	if resp.ResetDisplay == "" {
		t.Fatal("reset_display empty after Get recompute")
	}
}
```
