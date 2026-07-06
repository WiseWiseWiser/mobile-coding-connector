## Expected Output

Backup completes quietly or prints a short summary; primary artifact is the archive file.

## Expected

1. Exit code 0.
2. `OutputPath` exists and begins with xz magic bytes.
3. Archive lists `manifest.json`, `.bashrc`, `.ai-critic/ai-models.json`, `.cargo/config.toml`.
4. Archive omits `.cache/junk`, `.npm/x`, `.cargo/registry/db`, and `Projects/visible.txt`.
5. Symlink `.local/bin/tool-link` is archived (not followed).
6. `manifest.json` parses as JSON and references server home / exclusions.

## Side Effects

Creates `stream-backup.tar.xz` under `agentHome`.

## Errors

- Missing archive or invalid xz/tar layout.
- Excluded member present in archive listing.

## Exit Code

0.

```go
import (
	"encoding/json"
	"os"
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
		"manifest.json",
		".bashrc",
		".ai-critic/ai-models.json",
		".cargo/config.toml",
		".local/bin/tool-link",
	} {
		if !memberListContains(members, want) {
			t.Fatalf("archive missing %q; members=%v", want, members)
		}
	}
	for _, absent := range []string{
		".cache/junk",
		".npm/x/package.json",
		".cargo/registry/db/idx",
		"Projects/visible.txt",
	} {
		if memberListContains(members, absent) {
			t.Fatalf("archive unexpectedly contains excluded %q", absent)
		}
	}

	raw := tarXZExtractFile(t, resp.BackupPath, "manifest.json")
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("manifest.json invalid: %v\n%s", err, raw)
	}
	for _, key := range []string{"version", "created_at", "home", "excluded"} {
		if _, ok := manifest[key]; !ok {
			t.Fatalf("manifest missing %q: %v", key, manifest)
		}
	}

	bashrc := tarXZExtractFile(t, resp.BackupPath, ".bashrc")
	if string(bashrc) != "export FAKE=1\n" {
		t.Fatalf(".bashrc content mismatch: %q", bashrc)
	}
}
```