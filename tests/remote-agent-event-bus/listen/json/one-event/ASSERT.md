## Expected

1. Exit code 0.
2. Stdout contains exactly one non-empty JSON line (after optional blank trim) with the event type and id.
3. Stdout has **no** ANSI escape sequences.
4. JSON object unmarshals to shared Event shape (`type`, `id`).

## Errors

- Human pretty lines instead of JSON.
- ANSI colors on `--json` path.
- Missing type/id fields.

## Exit Code

0.

```go
import (
	"encoding/json"
	"strings"
	"testing"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
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
	if strings.Contains(resp.Stdout, "\x1b[") {
		t.Fatalf("--json stdout must not contain ANSI; got:\n%s", resp.Stdout)
	}
	// Prefer no ANSI on stderr either for pure JSON mode event path.
	if strings.Contains(resp.Stderr, "\x1b[") {
		t.Fatalf("--json stderr must not contain ANSI; got:\n%s", resp.Stderr)
	}

	line := firstNonEmptyLine(resp.Stdout)
	if line == "" {
		t.Fatalf("expected one JSON line; stdout:\n%s", resp.Stdout)
	}
	var ev sharedeb.Event
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		t.Fatalf("stdout line is not JSON Event: %v\nline=%q", err, line)
	}
	if ev.Type != fixtureTypeSeatalk {
		t.Fatalf("json type = %q, want %q", ev.Type, fixtureTypeSeatalk)
	}
	// Seeded id must round-trip (hub preserves provided id).
	if ev.ID != fixtureEventID1 {
		t.Fatalf("json id = %q, want %q", ev.ID, fixtureEventID1)
	}
}

func firstNonEmptyLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			return ln
		}
	}
	return ""
}
```
