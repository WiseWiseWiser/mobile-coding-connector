## Expected Output

Backup completes; archive lists injected `.backup/` meta members.

## Expected

1. Exit code 0.
2. Archive contains `.backup/config.json`, `.backup/installed.json`, `.backup/ENV`.
3. Archive contains `.backup/config.json.machine.bak` with seeded pre-backup content.
4. Archive `.backup/config.json` parses as effective exclusion config (`version` `1.0`).
5. Archive `.backup/installed.json` parses with `captured_at` and `tools` fields.

## Side Effects

Creates `backup-meta.tar.xz` under `agentHome`.

## Errors

- Missing meta members or wrong machine.bak content.

## Exit Code

0.

```go
import (
	"encoding/json"
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
	if resp.BackupPath == "" {
		t.Fatal("BackupPath empty")
	}
	if _, err := os.Stat(resp.BackupPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	archiveHasXZMagic(t, resp.BackupPath)
	members := tarXZListMembers(t, resp.BackupPath)
	for _, want := range []string{
		".backup/config.json",
		".backup/installed.json",
		".backup/ENV",
		".backup/config.json.machine.bak",
	} {
		if !memberListContains(members, want) {
			t.Fatalf("archive missing %q; members=%v", want, members)
		}
	}

	cfgRaw := tarXZExtractFile(t, resp.BackupPath, ".backup/config.json")
	cfg := parseExclusionConfigJSON(t, cfgRaw)
	if cfg.Version != "1.0" {
		t.Fatalf("archive config version = %q, want 1.0", cfg.Version)
	}
	if len(cfg.ExcludePaths) == 0 {
		t.Fatal("archive config exclude_paths empty")
	}

	bakRaw := tarXZExtractFile(t, resp.BackupPath, ".backup/config.json.machine.bak")
	if string(bakRaw) != seededBackupMetaJSON {
		t.Fatalf("machine.bak content = %q, want %q", bakRaw, seededBackupMetaJSON)
	}
	if strings.Contains(string(cfgRaw), "pre-backup-old") {
		t.Fatalf("archive config.json should be effective config, not seeded snapshot")
	}

	installedRaw := tarXZExtractFile(t, resp.BackupPath, ".backup/installed.json")
	var installed struct {
		CapturedAt string `json:"captured_at"`
		Tools      []any  `json:"tools"`
	}
	if err := json.Unmarshal(installedRaw, &installed); err != nil {
		t.Fatalf("installed.json invalid: %v\n%s", err, installedRaw)
	}
	if installed.CapturedAt == "" {
		t.Fatal("installed.json missing captured_at")
	}
	if installed.Tools == nil {
		t.Fatal("installed.json missing tools array")
	}
}
```