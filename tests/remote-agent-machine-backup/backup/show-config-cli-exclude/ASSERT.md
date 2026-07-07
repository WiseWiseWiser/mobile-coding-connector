## Expected Output

Stdout effective JSON includes `.knowledge-index` with reason `user excluded` and trailing newline.

## Expected

1. Exit code 0.
2. Stdout parses as effective exclusion config with trailing newline.
3. Effective `exclude_paths` includes `.knowledge-index` with reason `user excluded`.
4. No backup archive created.

## Side Effects

None (preview only; no persisted backup-config change).

## Errors

- CLI `--exclude` ignored (path missing from effective JSON).
- Reason is not `user excluded`.
- Unexpected backup archive on disk.

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
	assertEffectiveExcludeReason(t, cfg, ".knowledge-index", "user excluded")

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