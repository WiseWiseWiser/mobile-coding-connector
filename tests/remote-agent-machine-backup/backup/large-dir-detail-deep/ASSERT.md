## Expected Output

Flat `LARGE DIR DETAIL` lists nested `.deep-test/nested-big` and parent `.deep-test`
alongside `.big-test` rows; small sibling and builtin-excluded `.cache` are absent.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. `LARGE DIR DETAIL` is a flat sorted list (`> <rel-path>  <size>` per row).
4. Detail includes `.deep-test/nested-big` and `.deep-test` (both ≥ 10 MB).
5. Detail includes `.big-test`, `.big-test/child-a`, and `.big-test/child-b`.
6. Detail omits `.deep-test/small` (under 10 MB) and `.cache` (excluded).
7. Stream phase does not contain `LARGE DIR DETAIL` (summary only).
8. No backup archive file is created.

## Side Effects

None.

## Errors

- Nested path missing from detail (level-1-only scan).
- Excluded `.cache` listed in detail.
- Block-style nested headers instead of flat rows.

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

	assertLargeDirDetailHasPaths(t, resp.Combined,
		".big-test",
		".big-test/child-a",
		".big-test/child-b",
		".deep-test",
		".deep-test/nested-big",
	)
	assertLargeDirDetailLacksPaths(t, resp.Combined, ".deep-test/small", ".cache")

	streamPart := resp.Combined
	if idx := strings.Index(resp.Combined, "dry-run: machine backup plan"); idx >= 0 {
		streamPart = resp.Combined[:idx]
	}
	if strings.Contains(streamPart, "LARGE DIR DETAIL:") {
		t.Fatalf("stream phase must not contain LARGE DIR DETAIL; stream:\n%s", streamPart)
	}

	if resp.BackupPath != "" {
		if _, err := os.Stat(resp.BackupPath); err == nil {
			t.Fatalf("unexpected backup file %s from dry-run", resp.BackupPath)
		}
	}
}
```