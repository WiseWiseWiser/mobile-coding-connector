## Expected Output

Stdout is indented JSON with `version` `1.1` and extended `exclude_paths` including
special rules (`**(binary)`, `**/*.log`, `**/upload-chunks`) and five new path prefixes.
Built-in config no longer lists `.config/git-fetch-skill/data`,
`.config/confluence-fetch-skill/data`, or `.knowledge-index`.

## Expected

1. Exit code 0.
2. Stdout parses as exclusion config with `version` `1.1`.
3. `exclude_paths` includes all new special rules and five path-prefix entries with exact reasons.
4. `exclude_paths` omits `.config/git-fetch-skill/data`, `.config/confluence-fetch-skill/data`, and `.knowledge-index`.
5. Stdout ends with a trailing newline after the JSON.
6. No backup archive file is created under `agentHome`.

## Side Effects

None.

## Errors

- Invalid JSON, wrong version, or missing extended entries.
- Unexpected archive file on disk.

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

	stdout := resp.Stdout
	if stdout == "" || stdout[len(stdout)-1] != '\n' {
		t.Fatalf("stdout must end with newline; got len=%d tail=%q", len(stdout), stdout)
	}

	cfg := parseExclusionConfigJSON(t, []byte(strings.TrimSpace(stdout)))
	assertExclusionConfigV11(t, cfg)
	for _, removed := range []string{
		".config/git-fetch-skill/data",
		".config/confluence-fetch-skill/data",
		".knowledge-index",
	} {
		if _, ok := exclusionConfigHasPath(cfg, removed); ok {
			t.Fatalf("exclude_paths still contains reverted path %q", removed)
		}
	}

	matches, _ := filepath.Glob(filepath.Join(resp.AgentHome, "machine-backup-*.tar.xz"))
	if len(matches) > 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
	if resp.BackupPath != "" {
		if _, err := os.Stat(resp.BackupPath); err == nil {
			t.Fatalf("unexpected backup file %s", resp.BackupPath)
		}
	}
}
```