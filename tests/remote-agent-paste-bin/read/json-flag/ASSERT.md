## Expected Output

```json
{"content":"line1\nline2\nemoji🎉","updated_at":"2026-07-14T08:30:00Z"}
```

## Expected

1. Exit code 0.
2. Stdout is valid JSON with `content` and `updated_at` matching seeded scratch.
3. Stdout contains no ANSI escape sequences.
4. Stderr empty (no preview/meta lines on `--json` read).

## Side Effects

Scratch API unchanged.

## Errors

- Human-readable content without JSON wrapper on stdout.
- ANSI color codes in `--json` output.
- Missing `content` or `updated_at` fields.

## Exit Code

0.

```go
import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if strings.Contains(resp.Stdout, "\x1b[") {
		t.Fatalf("--json stdout must not contain ANSI; got:\n%s", resp.Stdout)
	}
	if resp.Stderr != "" {
		t.Fatalf("expected silent stderr for --json read; got:\n%s", resp.Stderr)
	}

	var payload ScratchEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, resp.Stdout)
	}
	if payload.Content != seededUTF8Content {
		t.Fatalf("json content = %q, want %q", payload.Content, seededUTF8Content)
	}
	if payload.UpdatedAt != seededMetaUpdatedAt {
		t.Fatalf("json updated_at = %q, want %q", payload.UpdatedAt, seededMetaUpdatedAt)
	}

	assert.Output(t, strings.TrimSpace(resp.Stdout), `---
version: 2
---
{"content":"line1\nline2\nemoji🎉","updated_at":"2026-07-14T08:30:00Z"}`)
}
```