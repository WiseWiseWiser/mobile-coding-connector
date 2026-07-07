## Expected Output

Summary includes `codex sessions:` / `codex skills:` because `.codex` exists.
Summary omits `grok sessions:`, `grok projects:`, and `grok skills:` because
`.grok` is absent from server home.

## Expected

1. Exit code 0.
2. Combined output contains `codex sessions:` and `codex skills:`.
3. Combined output does not contain `grok sessions:`, `grok projects:`, or `grok skills:`.
4. No `> .grok` entry block appears.

## Side Effects

None.

## Errors

- Grok summary lines present without `.grok` dir.
- Missing codex summary when `.codex` seeded.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	combinedHasAll(t, resp.Combined,
		"> .codex",
		"codex sessions:",
		"codex skills:",
	)
	combinedHasNone(t, resp.Combined,
		"grok sessions:",
		"grok projects:",
		"grok skills:",
	)

	if strings.Contains(resp.Combined, "> .grok") {
		t.Fatalf("unexpected .grok entry block when .grok absent:\n%s", resp.Combined)
	}
}
```