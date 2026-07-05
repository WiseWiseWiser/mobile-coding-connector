---
label: slow && real-codex && negative
explanation: real codex CLI via tty-watch; manual early /status\r script must not show usage fields
---

## Expected

The user-script pattern must **not** surface parseable usage (documents anti-pattern):

1. `StatusFieldsSeen` is false.
2. `StatusReadySecs` is `0`.

## Errors

- `StatusFieldsSeen=true` (would falsely imply early `/status\r` is sufficient).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusFieldsSeen {
		t.Fatalf("status fields appeared unexpectedly; transcript:\n%s", resp.TTYWatchTranscript)
	}
	if resp.StatusReadySecs != 0 {
		t.Fatalf("status_ready_secs = %d, want 0", resp.StatusReadySecs)
	}
}
```