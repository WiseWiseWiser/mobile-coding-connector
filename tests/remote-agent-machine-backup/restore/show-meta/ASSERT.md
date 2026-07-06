## Expected Output

Stdout prints `=== .backup/ENV ===` and `=== .backup/installed.json ===` sections
with archive content; does not print `config.json` or `*.machine.bak`.

## Expected

1. Exit code 0.
2. Combined output contains `=== .backup/installed.json ===` and `captured_at`.
3. Combined output contains `=== .backup/ENV ===` and at least one `KEY=VALUE` line.
4. Combined output does not contain `=== .backup/config.json ===`.
5. Combined output does not contain `.machine.bak`.

## Side Effects

None (read-only archive inspection).

## Errors

- Missing meta sections or config.json leaked into output.

## Exit Code

0.

```go
import (
	"encoding/json"
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
	if resp.BackupPath == "" {
		t.Fatal("prereq BackupPath empty")
	}

	combinedHasAll(t, resp.Combined,
		"=== .backup/installed.json ===",
		"captured_at",
		"=== .backup/ENV ===",
	)
	combinedHasNone(t, resp.Combined,
		"=== .backup/config.json ===",
		".machine.bak",
	)

	installedRaw := tarXZExtractFile(t, resp.BackupPath, ".backup/installed.json")
	var installed struct {
		CapturedAt string `json:"captured_at"`
	}
	if err := json.Unmarshal(installedRaw, &installed); err != nil {
		t.Fatalf("archive installed.json invalid: %v", err)
	}
	if !strings.Contains(resp.Combined, installed.CapturedAt) {
		t.Fatalf("stdout missing captured_at %q; got:\n%s", installed.CapturedAt, resp.Combined)
	}

	envRaw := tarXZExtractFile(t, resp.BackupPath, ".backup/ENV")
	firstLine := strings.TrimSpace(strings.Split(string(envRaw), "\n")[0])
	if firstLine == "" {
		t.Fatal("archive ENV empty")
	}
	if !strings.Contains(resp.Combined, firstLine) {
		t.Fatalf("stdout missing ENV line %q; got:\n%s", firstLine, resp.Combined)
	}
}
```