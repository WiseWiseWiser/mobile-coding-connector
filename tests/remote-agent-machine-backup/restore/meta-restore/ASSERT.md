## Expected Output

Restore apply streams CLASSIFYING (and APPLYING for non-skip entries); server
`~/.backup/config.json` ends with seeded pre-backup content.

## Expected

1. Exit code 0.
2. Combined output has `CLASSIFYING:` and `APPLYING:` (apply stream).
3. `serverHome/.backup/config.json` equals seeded pre-backup JSON.
4. `serverHome/.backup/installed.json` does not exist (meta snapshot not applied).
5. `serverHome/.backup/ENV` does not exist (meta snapshot not applied).
6. Archive effective `.backup/config.json` differs from restored server config.

## Side Effects

Restores `~/.backup/config.json` from `.backup/config.json.machine.bak`; does not
write current meta snapshots from archive.

## Errors

- Server config still wiped JSON or matches archive effective config.
- Meta snapshots written to server home.

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
	if resp.BackupPath == "" {
		t.Fatal("prereq BackupPath empty")
	}

	assertRestoreStreamSections(t, resp.Combined, true)

	got := readServerFile(t, resp.ServerHome, ".backup/config.json")
	if got != seededBackupMetaJSON {
		t.Fatalf("server .backup/config.json = %q, want seeded %q", got, seededBackupMetaJSON)
	}

	for _, absent := range []string{".backup/installed.json", ".backup/ENV"} {
		full := filepath.Join(resp.ServerHome, filepath.FromSlash(absent))
		if _, err := os.Stat(full); err == nil {
			t.Fatalf("meta snapshot %s should not exist on server after restore", absent)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", full, err)
		}
	}

	archiveCfg := string(tarXZExtractFile(t, resp.BackupPath, ".backup/config.json"))
	if strings.Contains(archiveCfg, "pre-backup-old") {
		t.Fatalf("archive effective config should not contain seeded marker")
	}
	if strings.Contains(got, `"version"`) && strings.Contains(got, `"exclude_paths"`) {
		t.Fatalf("restored config looks like effective archive config, not machine.bak")
	}
}
```