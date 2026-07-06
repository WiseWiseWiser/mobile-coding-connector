## Expected Output

Stdout is indented JSON with `version` and `exclude_paths` array; no archive written.

## Expected

1. Exit code 0.
2. Stdout parses as exclusion config with `version` `1.0`.
3. `exclude_paths` includes `.cache` with a non-empty `reason`.
4. No backup archive file is created under `agentHome`.

## Side Effects

None.

## Errors

- Invalid JSON or missing `exclude_paths`.
- Unexpected archive file on disk.

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

	cfg := parseExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	if cfg.Version != "1.0" {
		t.Fatalf("version = %q, want 1.0", cfg.Version)
	}
	if len(cfg.ExcludePaths) == 0 {
		t.Fatal("exclude_paths empty")
	}
	foundCache := false
	for _, e := range cfg.ExcludePaths {
		if e.Path == ".cache" {
			foundCache = true
			if strings.TrimSpace(e.Reason) == "" {
				t.Fatal(".cache exclusion missing reason")
			}
		}
	}
	if !foundCache {
		t.Fatalf("exclude_paths missing .cache: %+v", cfg.ExcludePaths)
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