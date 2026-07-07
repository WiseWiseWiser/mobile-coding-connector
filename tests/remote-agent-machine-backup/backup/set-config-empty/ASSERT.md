## Expected Output

Non-zero exit; stderr/stdout mentions missing set-config input.

## Expected

1. Non-zero exit code.
2. Combined output mentions `--set-config` and either `--exclude` or `--large-dir-threshold` (or states input is required).
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
		t.Fatalf("expected non-zero exit for bare --set-config; combined:\n%s", resp.Combined)
	}
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "set-config") {
		t.Fatalf("expected error mentioning set-config; got:\n%s", resp.Combined)
	}
	if !strings.Contains(lower, "exclude") && !strings.Contains(lower, "large-dir-threshold") && !strings.Contains(lower, "large_dir_threshold") {
		t.Fatalf("expected error mentioning required input flags; got:\n%s", resp.Combined)
	}

	if _, err := os.Stat(userBackupConfigPath(resp.ServerHome)); err == nil {
		t.Fatal("backup-config.json should not exist after failed set-config")
	}
}
```