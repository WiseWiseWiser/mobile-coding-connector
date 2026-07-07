## Expected Output

Stdout effective JSON: `.knowledge-hub` reason `from user config`; `.knowledge-index` keeps manual reason.

## Expected

1. Exit code 0.
2. Stdout parses as effective config with trailing newline.
3. Effective `exclude_paths` includes `.knowledge-hub` with reason `from user config`.
4. Effective `exclude_paths` includes `.knowledge-index` with reason `knowledge index cache`.
5. No backup archive created.

## Side Effects

Persists and patches backup-config.json via prereq set-config + post patch.

## Errors

- CLI-set path shows `user excluded` instead of `from user config`.
- Hand-edited reason overwritten.

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

	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("stdout missing trailing newline; got %q", resp.Stdout)
	}
	cfg := parseEffectiveExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	assertEffectiveExcludeReason(t, cfg, ".knowledge-hub", "from user config")
	assertEffectiveExcludeReason(t, cfg, ".knowledge-index", "knowledge index cache")

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
	if resp.BackupPath != "" {
		if _, err := os.Stat(resp.BackupPath); err == nil {
			t.Fatalf("unexpected backup file %s", resp.BackupPath)
		}
	}
}
```