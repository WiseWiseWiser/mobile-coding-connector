## Expected Output

50 MB `.big-test` stays in DOT DIRS but omits `LARGE SIZE` when persisted threshold is 100 MB;
flat `LARGE DIR DETAIL` still lists ≥ 10 MB included dirs.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan` and `.big-test`.
3. Summary DOT DIRS row for `.big-test` does not include `LARGE SIZE`.
4. `LARGE DIR DETAIL` flat list still includes `.big-test` and its large children.
5. No backup archive file is created.

## Side Effects

Persists `large_dir_threshold` via prereq set-config.

## Errors

- `LARGE SIZE` present despite persisted 100 MB threshold.
- Missing `LARGE DIR DETAIL` for ≥ 10 MB dirs.
- Default 40 MB threshold still applied for LARGE SIZE flag.

## Exit Code

0.

```go
import (
	"os"
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
	if !strings.Contains(resp.Combined, "dry-run: machine backup plan") {
		t.Fatalf("missing backup plan summary; got:\n%s", resp.Combined)
	}
	if !strings.Contains(resp.Combined, ".big-test") {
		t.Fatalf("missing .big-test in output; got:\n%s", resp.Combined)
	}

	assertSummaryDirLacksLargeSize(t, resp.Combined, ".big-test")
	assertLargeDirDetailHasPaths(t, resp.Combined, ".big-test", ".big-test/child-a", ".big-test/child-b")

	if resp.BackupPath != "" {
		if _, err := os.Stat(resp.BackupPath); err == nil {
			t.Fatalf("unexpected backup file %s from dry-run", resp.BackupPath)
		}
	}
}
```