## Expected

1. Exit code 0 (MaxEvents stop).
2. Stdout contains connected line with `server=` and the hub base URL (or host:port).
3. Stdout contains event type `seatalk.message.received`.
4. Human path may use ANSI (green connected); type string must be visible.

## Errors

- No connect line.
- Event type missing.
- Non-zero exit when event was received.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	if !strings.Contains(out, "connected") {
		t.Fatalf("expected connected line; stdout:\n%s", out)
	}
	if !strings.Contains(out, "server=") {
		t.Fatalf("expected server= in connected line; stdout:\n%s", out)
	}
	// Server URL host:port should appear (allow ANSI around connected).
	if resp.ServerURL != "" {
		host := strings.TrimPrefix(strings.TrimPrefix(resp.ServerURL, "http://"), "https://")
		if !strings.Contains(out, host) && !strings.Contains(out, resp.ServerURL) {
			t.Fatalf("connected line should reference server %q; stdout:\n%s", resp.ServerURL, out)
		}
	}
	if !strings.Contains(out, fixtureTypeSeatalk) {
		t.Fatalf("expected event type %q; stdout:\n%s", fixtureTypeSeatalk, out)
	}
}
```
