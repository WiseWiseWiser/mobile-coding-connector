## Expected Output

Dry-run plan lists `.cache` under DOT DIRS (included) rather than EXCLUDED with
a built-in cache reason.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. DOT DIRS section includes `.cache` before the EXCLUDED section.
4. EXCLUDED section does not list `.cache` with `temporary application cache`.
5. No backup archive file is created.

## Side Effects

None.

## Errors

- `.cache` only appears as excluded.
- Missing dry-run summary.

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

	idxDirs := strings.Index(resp.Combined, "DOT DIRS")
	idxExcluded := strings.Index(resp.Combined, "EXCLUDED")
	if idxDirs < 0 || idxExcluded < 0 || idxDirs >= idxExcluded {
		t.Fatalf("missing DOT DIRS / EXCLUDED sections; got:\n%s", resp.Combined)
	}
	dotDirs := resp.Combined[idxDirs:idxExcluded]
	if !strings.Contains(dotDirs, ".cache") {
		t.Fatalf(".cache missing from DOT DIRS; dot dirs section:\n%s", dotDirs)
	}

	excluded := resp.Combined[idxExcluded:]
	if strings.Contains(excluded, ".cache") && strings.Contains(excluded, "temporary application cache") {
		t.Fatalf(".cache still excluded after --include; excluded section:\n%s", excluded)
	}

	if resp.BackupPath != "" {
		if _, err := os.Stat(resp.BackupPath); err == nil {
			t.Fatalf("unexpected backup file %s from dry-run", resp.BackupPath)
		}
	}
}
```