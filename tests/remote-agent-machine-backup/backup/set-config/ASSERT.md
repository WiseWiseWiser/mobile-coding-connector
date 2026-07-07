## Expected Output

Exit 0; stdout is effective merged JSON with trailing newline; persisted file omits CLI-set exclude reason.

## Expected

1. Exit code 0.
2. Stdout parses as effective exclusion config with `version` `1.1` and trailing newline.
3. `backup-config.json` exists under server home with `version` `1.1`.
4. `exclude_paths` includes `.knowledge-hub` with empty or omitted `reason` (not `user excluded`).
5. No backup archive created.

## Side Effects

Writes `.ai-critic/backup-config.json` on server home.

## Errors

- Missing or invalid persisted config file.
- Persisted entry has `user excluded` reason.
- Unexpected backup archive.

## Exit Code

0.

```go
import (
	"os"
	"path/filepath"
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

	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("stdout missing trailing newline; got %q", resp.Stdout)
	}
	effective := parseEffectiveExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))
	if effective.Version != "1.1" {
		t.Fatalf("stdout version = %q, want 1.1", effective.Version)
	}

	raw, err := os.ReadFile(userBackupConfigPath(resp.ServerHome))
	if err != nil {
		t.Fatalf("read backup-config.json: %v", err)
	}
	assertPersistedExcludeEmptyReason(t, raw, ".knowledge-hub")

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
}
```