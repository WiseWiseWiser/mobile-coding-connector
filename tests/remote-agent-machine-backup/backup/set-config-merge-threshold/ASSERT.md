## Expected Output

Exit 0; persisted file contains `.knowledge-hub`, `.docker`, and `large_dir_threshold` `50MB`.

## Expected

1. Exit code 0.
2. Stdout parses as effective exclusion config with trailing newline.
3. `backup-config.json` includes both exclude paths with empty/omitted reasons.
4. `large_dir_threshold` remains `50MB` after exclude-only set-config.
5. No backup archive created.

## Side Effects

Writes merged `.ai-critic/backup-config.json` on server home.

## Errors

- Missing either persisted exclude path.
- Threshold wiped or changed by exclude-only set-config.
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
	if effective.LargeDirThreshold != "50MB" {
		t.Fatalf("stdout large_dir_threshold = %q, want 50MB", effective.LargeDirThreshold)
	}

	raw, err := os.ReadFile(userBackupConfigPath(resp.ServerHome))
	if err != nil {
		t.Fatalf("read backup-config.json: %v", err)
	}
	assertPersistedExcludeEmptyReason(t, raw, ".knowledge-hub")
	assertPersistedExcludeEmptyReason(t, raw, ".docker")

	persisted := parseUserBackupConfigJSON(t, raw)
	if persisted.LargeDirThreshold != "50MB" {
		t.Fatalf("persisted large_dir_threshold = %q, want 50MB", persisted.LargeDirThreshold)
	}

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
}
```