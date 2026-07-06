## Expected Output

Stream phase prints DOT FILES / DOT DIRS / EXCLUDED with per-entry sizes (B, KB,
or MB). Summary phase ends with `dry-run: machine backup plan` and rollups.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. Combined output includes at least one size token (` B`, ` KB`, or ` MB`).
4. Combined output mentions `.bashrc` and `.ai-critic`.
5. Combined output lists built-in exclusions: `.cache`, `.npm`, `.cargo/registry`.
6. EXCLUDED section for `.cache` includes a reason token (e.g. `temporary application cache` or `cache`).
7. Combined output does not claim `.cache/junk` or `.npm/x` are included.
8. No backup archive file is created under `agentHome`.

## Side Effects

None (no archive write).

## Errors

- Archive file appears on disk.
- Excluded trees listed as included.
- Missing backup plan summary or size tokens.

## Exit Code

0.

```go
import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var backupSizeToken = regexp.MustCompile(`\d+(\.\d+)?\s*(B|KB|MB)\b`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	if !strings.Contains(resp.Combined, "dry-run: machine backup plan") {
		t.Fatalf("missing backup plan summary; got:\n%s", resp.Combined)
	}
	if !backupSizeToken.MatchString(resp.Combined) {
		t.Fatalf("missing size token (B/KB/MB); got:\n%s", resp.Combined)
	}

	combinedHasAll(t, resp.Combined,
		".bashrc",
		".ai-critic",
		".cache",
		".npm",
	)
	assertCacheExclusionReason(t, resp.Combined)
	combinedHasNone(t, resp.Combined, ".cache/junk", ".npm/x", ".cargo/registry/db")

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected default backup files: %v", matches)
	}
	if resp.BackupPath != "" {
		if _, err := os.Stat(resp.BackupPath); err == nil {
			t.Fatalf("unexpected backup file %s from dry-run", resp.BackupPath)
		}
	}

	// Non-dot top-level path must not appear as included.
	if strings.Contains(resp.Combined, "Projects/visible.txt") &&
		!strings.Contains(strings.ToLower(resp.Combined), "excluded") {
		t.Fatalf("non-dot Projects/ should not be included without exclusion context")
	}
}
```