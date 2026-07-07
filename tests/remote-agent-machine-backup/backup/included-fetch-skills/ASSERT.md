## Expected Output

Dry-run plan includes files under `.config/git-fetch-skill/data`,
`.config/confluence-fetch-skill/data`, and `.knowledge-index`; they do not appear
under EXCLUDED with the former built-in reasons.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. Combined output mentions seeded paths:
   - `.config/git-fetch-skill/data/cache`
   - `.config/confluence-fetch-skill/data/note`
   - `.knowledge-index/agents.json`
4. EXCLUDED section does not list the reverted path-prefix rules
   (`git-fetch-skill`, `confluence-fetch-skill data cache`, `knowledge index cache`).
5. No backup archive file is created.

## Side Effects

None.

## Errors

- Reverted paths missing from output or listed as excluded.

## Exit Code

0.

```go
import (
	"os"
	"strings"
	"testing"
)

func assertPathsNotInExcluded(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := excludedSection(combined)
	if section == "" {
		return
	}
	for _, p := range paths {
		if strings.Contains(section, p) {
			t.Fatalf("path %q unexpectedly in EXCLUDED; section:\n%s", p, section)
		}
	}
}

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

	wantPaths := []string{
		".config/git-fetch-skill/data/cache",
		".config/confluence-fetch-skill/data/note",
		".knowledge-index/agents.json",
	}
	combinedHasAll(t, resp.Combined, wantPaths...)
	assertPathsNotInExcluded(t, resp.Combined, wantPaths...)

	excluded := excludedSection(resp.Combined)
	for _, reason := range []string{
		"git-fetch-skill data cache",
		"confluence-fetch-skill data cache",
		"knowledge index cache",
	} {
		if strings.Contains(excluded, reason) {
			t.Fatalf("EXCLUDED still lists reverted reason %q; section:\n%s", reason, excluded)
		}
	}

	if resp.BackupPath != "" {
		if _, err := os.Stat(resp.BackupPath); err == nil {
			t.Fatalf("unexpected backup file %s from dry-run", resp.BackupPath)
		}
	}
}
```