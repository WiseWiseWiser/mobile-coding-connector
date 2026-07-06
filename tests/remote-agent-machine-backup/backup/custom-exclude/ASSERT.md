## Expected Output

Backup succeeds; `.docker` appears among exclusions and is absent from the archive.

## Expected

1. Exit code 0.
2. Archive exists with xz magic.
3. `.docker/config` is not a tar member.
4. `manifest.json` `excluded` list includes `.docker`.
5. `.bashrc` remains included.

## Side Effects

Creates `custom-exclude.tar.xz` under `agentHome`.

## Errors

- `.docker/config` present in archive.
- Custom exclude not recorded in manifest.

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
	if memberListContains(members, ".docker/config") {
		t.Fatalf("archive should exclude .docker/config; members=%v", members)
	}
	if !memberListContains(members, ".bashrc") {
		t.Fatalf("archive missing .bashrc; members=%v", members)
	}

	raw := tarXZExtractFile(t, resp.BackupPath, "manifest.json")
	var manifest struct {
		Excluded []string `json:"excluded"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("manifest.json invalid: %v", err)
	}
	foundDocker := false
	for _, ex := range manifest.Excluded {
		if ex == ".docker" || strings.HasPrefix(ex, ".docker/") {
			foundDocker = true
			break
		}
	}
	if !foundDocker {
		t.Fatalf("manifest excluded missing .docker: %v", manifest.Excluded)
	}
}
```