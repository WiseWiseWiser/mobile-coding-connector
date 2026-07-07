## Expected Output

EXCLUDED section header shows aggregate paths, files, and bytes skipped. Per-rule table
lists RULE / FILES / SIZE / REASON columns sorted by SIZE descending. `.cache` row
aggregates both cache files (>= 2 files, >= 1 KB); `**/*.log` row shows at least one
file. `.cache/junk` does not appear in DOT FILES.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. EXCLUDED header matches `EXCLUDED (N paths, F files, <size>)` with a size token.
4. EXCLUDED section includes column headers `RULE` and `FILES`.
5. `.cache` rule row shows FILES >= 2 and SIZE >= 1 KB with a cache reason token.
6. `**/*.log` (or `*.log`) rule row shows FILES >= 1.
7. `.cache` rule row appears before `**/*.log` row (descending SIZE order).
8. DOT FILES does not include `.cache/junk`.
9. DOT FILES includes `.bashrc` (included control path).

## Side Effects

None.

## Errors

- Missing EXCLUDED aggregate header or per-rule table columns.
- Per-file excluded paths listed in DOT FILES.
- Rules not sorted by SIZE descending when `.cache` is larger than log rule.

## Exit Code

0.

```go
import (
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

	assertExcludedStatsHeader(t, resp.Combined)
	assertExcludedTableHeaders(t, resp.Combined)
	assertCacheExclusionReason(t, resp.Combined)

	assertExcludedRuleFilesAtLeast(t, resp.Combined, ".cache", 2)
	assertExcludedRuleHasSizeKB(t, resp.Combined, ".cache")

	logRule := "**/*.log"
	if !strings.Contains(resp.Combined, logRule) {
		logRule = "*.log"
	}
	assertExcludedRuleFilesAtLeast(t, resp.Combined, logRule, 1)
	assertExcludedRuleBefore(t, resp.Combined, ".cache", logRule)

	assertDotFilesExcludes(t, resp.Combined, ".cache/junk")
	assertDotFilesIncludes(t, resp.Combined, ".bashrc")
}
```