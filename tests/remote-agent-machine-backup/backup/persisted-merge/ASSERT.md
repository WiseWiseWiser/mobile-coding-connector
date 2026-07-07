## Expected Output

Dry-run succeeds; persisted excludes appear in EXCLUDED section; follow-up show-config displays `from user config`.

## Expected

1. Exit code 0.
2. Combined output mentions `.knowledge-hub` in EXCLUDED section.
3. DOT DIRS does not list `.knowledge-hub` or `.knowledge-index`.
4. Follow-up `--show-config` stdout shows `.knowledge-hub` with reason `from user config`.
5. No archive written.

## Side Effects

Persists backup-config.json via prereq set-config.

## Errors

- Persisted paths still included in plan.
- Missing EXCLUDED mention for user path.
- Show-config effective reason is not `from user config`.

## Exit Code

0.

```go
import (
	"os"
	"path/filepath"
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

	combined := resp.Combined
	if !strings.Contains(combined, ".knowledge-hub") {
		t.Fatalf("EXCLUDED/summary should mention .knowledge-hub; got:\n%s", combined)
	}
	assertDotDirsExcludes(t, combined, ".knowledge-hub", ".knowledge-index")

	if !strings.HasSuffix(resp.FollowUpStdout, "\n") {
		t.Fatalf("follow-up show-config stdout missing trailing newline; got %q", resp.FollowUpStdout)
	}
	cfg := parseEffectiveExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.FollowUpStdout)))
	assertEffectiveExcludeReason(t, cfg, ".knowledge-hub", "from user config")

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
}
```