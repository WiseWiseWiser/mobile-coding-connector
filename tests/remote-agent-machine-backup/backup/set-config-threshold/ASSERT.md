## Expected Output

Exit 0; persisted file contains `large_dir_threshold` `100MB` and prior `.knowledge-hub` exclude.

## Expected

1. Exit code 0.
2. Stdout parses as effective config with trailing newline and `large_dir_threshold` `100MB`.
3. `backup-config.json` contains `large_dir_threshold` `100MB`, `version` `1.1`, and `.knowledge-hub` in `exclude_paths`.
4. Prereq exclude path preserved after threshold-only set-config.
5. No backup archive created.

## Side Effects

Writes `.ai-critic/backup-config.json` on server home.

## Errors

- Missing `large_dir_threshold` in persisted or stdout JSON.
- Unexpected backup archive.

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
	effective := parseEffectiveExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	if effective.LargeDirThreshold != "100MB" {
		t.Fatalf("stdout large_dir_threshold = %q, want 100MB", effective.LargeDirThreshold)
	}

	raw, err := os.ReadFile(userBackupConfigPath(resp.ServerHome))
	if err != nil {
		t.Fatalf("read backup-config.json: %v", err)
	}
	persisted := parseUserBackupConfigJSON(t, raw)
	if persisted.LargeDirThreshold != "100MB" {
		t.Fatalf("persisted large_dir_threshold = %q, want 100MB", persisted.LargeDirThreshold)
	}
	if persisted.Version != "1.1" {
		t.Fatalf("persisted version = %q, want 1.1", persisted.Version)
	}
	assertPersistedExcludeEmptyReason(t, raw, ".knowledge-hub")

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
}
```