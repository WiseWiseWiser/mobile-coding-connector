## Expected Output

Summary flags `.big-test` with `LARGE SIZE` (plain text when stdout is piped),
lists DOT DIRS by size descending, and prints flat `LARGE DIR DETAIL` with parent
and child rows sorted by size descending.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. Summary DOT DIRS row for `.big-test` includes `LARGE SIZE`.
4. `.big-test` appears before `.small-test` in summary DOT DIRS (size-desc sort).
5. `LARGE DIR DETAIL` flat list includes `.big-test`, `.big-test/child-a`, and `.big-test/child-b` (size-desc).
6. Stream phase does not contain `LARGE DIR DETAIL` (summary only).
7. No backup archive file is created.

## Side Effects

None.

## Errors

- Missing `LARGE SIZE` on `.big-test`.
- Missing or empty `LARGE DIR DETAIL`.
- Block-style nested headers instead of flat rows.
- DOT DIRS not size-sorted.

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

	assertSummaryDirHasLargeSize(t, resp.Combined, ".big-test")
	assertDotDirsSortedBySizeDesc(t, resp.Combined)

	section := dotDirsSummarySection(resp.Combined)
	rows := parseDotDirSummaryRows(section)
	bigIdx, smallIdx := -1, -1
	for i, row := range rows {
		switch row.Path {
		case ".big-test":
			bigIdx = i
		case ".small-test":
			smallIdx = i
		}
	}
	if bigIdx < 0 || smallIdx < 0 {
		t.Fatalf("missing .big-test or .small-test in DOT DIRS summary; section:\n%s", section)
	}
	if bigIdx > smallIdx {
		t.Fatalf(".big-test should sort before .small-test (idx %d vs %d); section:\n%s", bigIdx, smallIdx, section)
	}

	detailRows := assertLargeDirDetailFlatSorted(t, resp.Combined)
	assertLargeDirDetailHasPaths(t, resp.Combined, ".big-test", ".big-test/child-a", ".big-test/child-b")
	if detailRows[0].Path != ".big-test" {
		t.Fatalf("largest detail row want .big-test, got %q; rows=%v", detailRows[0].Path, detailRows)
	}

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