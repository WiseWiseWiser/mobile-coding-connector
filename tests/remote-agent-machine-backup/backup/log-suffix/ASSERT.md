## Expected Output

Dry-run plan lists `**/*.log` (or log-file reason) under EXCLUDED; `.ai-critic/service.log`
is omitted from DOT FILES while `.ai-critic/config.json` remains included.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. EXCLUDED section mentions `*.log` or `log files`.
4. DOT FILES does not include `.ai-critic/service.log`.
5. DOT FILES includes `.ai-critic/config.json`.

## Side Effects

None.

## Errors

- `.log` file listed as included.
- Missing log-suffix exclusion rule in EXCLUDED.

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

	section := excludedSection(resp.Combined)
	if !strings.Contains(section, "*.log") && !strings.Contains(section, "log files") {
		t.Fatalf("EXCLUDED missing log suffix rule; section:\n%s", section)
	}

	assertDotFilesExcludes(t, resp.Combined, ".ai-critic/service.log")
	assertDotFilesIncludes(t, resp.Combined, ".ai-critic/config.json")
}
```