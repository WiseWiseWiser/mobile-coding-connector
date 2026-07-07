## Expected Output

Exit 0; persisted file contains both `.knowledge-hub` and `.docker` with empty/omitted reasons.

## Expected

1. Exit code 0.
2. Stdout parses as effective exclusion config with trailing newline.
3. `backup-config.json` includes `.knowledge-hub` and `.docker` in `exclude_paths` (empty or omitted reason).
4. Prior `.knowledge-hub` entry is preserved after second set-config.
5. No backup archive created.

## Side Effects

Writes merged `.ai-critic/backup-config.json` on server home.

## Errors

- Missing either persisted exclude path.
- Second set-config replaced prior excludes instead of merging.
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
	parseEffectiveExclusionConfigJSON(t, []byte(strings.TrimSpace(resp.Stdout)))

	raw, err := os.ReadFile(userBackupConfigPath(resp.ServerHome))
	if err != nil {
		t.Fatalf("read backup-config.json: %v", err)
	}
	assertPersistedExcludeEmptyReason(t, raw, ".knowledge-hub")
	assertPersistedExcludeEmptyReason(t, raw, ".docker")

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
}
```