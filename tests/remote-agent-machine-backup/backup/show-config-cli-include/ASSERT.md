## Expected Output

Stdout effective JSON omits `.cache` from `exclude_paths`; other built-in excludes remain.

## Expected

1. Exit code 0.
2. Stdout parses as effective exclusion config with trailing newline.
3. Effective `exclude_paths` does not include `.cache`.
4. Effective `exclude_paths` still includes another built-in path (e.g. `.npm`).
5. No backup archive created.

## Side Effects

Prereq persists `backup-config.json` with `.cache` exclude; main invocation is preview only.

## Errors

- `.cache` still present in effective `exclude_paths` after `--include`.
- Missing trailing newline.
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
	cfg := parseEffectiveExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	for _, e := range cfg.ExcludePaths {
		if e.Path == ".cache" {
			t.Fatalf("effective exclude_paths still contains .cache after --include: %+v", cfg.ExcludePaths)
		}
	}
	foundNPM := false
	for _, e := range cfg.ExcludePaths {
		if e.Path == ".npm" {
			foundNPM = true
			break
		}
	}
	if !foundNPM {
		t.Fatalf("effective exclude_paths missing built-in .npm; got %+v", cfg.ExcludePaths)
	}

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