## Expected Output

Non-zero exit; error states --set-config conflicts with --dry-run.

## Expected

1. Non-zero exit code.
2. Combined output mentions `--set-config` and `--dry-run` (or mutually exclusive).
3. No `backup-config.json` written on server home.

## Side Effects

None.

## Errors

- Process exits 0.
- Persisted config file created.

## Exit Code

Non-zero.

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
	if resp.ExitCode == 0 {
		t.Fatalf("expected non-zero exit for --set-config --dry-run; combined:\n%s", resp.Combined)
	}
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "set-config") {
		t.Fatalf("expected error mentioning set-config; got:\n%s", resp.Combined)
	}
	if !strings.Contains(lower, "dry-run") && !strings.Contains(lower, "mutually exclusive") {
		t.Fatalf("expected error mentioning dry-run or mutual exclusion; got:\n%s", resp.Combined)
	}

	if _, err := os.Stat(userBackupConfigPath(resp.ServerHome)); err == nil {
		t.Fatal("backup-config.json should not exist after failed set-config")
	}
}
```