## Expected Output

Backup archive includes `.live-and-love/imgs/photo.jpg`; user images are not
excluded by the binary rule.

## Expected

1. Exit code 0.
2. `OutputPath` exists with xz magic bytes.
3. Archive member list includes `.live-and-love/imgs/photo.jpg`.

## Side Effects

Creates `keep-images-backup.tar.xz` under `agentHome`.

## Errors

- JPEG image missing from archive.
- Archive invalid or unreadable.

## Exit Code

0.

```go
import (
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
	if !memberListContains(members, ".live-and-love/imgs/photo.jpg") {
		t.Fatalf("archive missing user image; members=%v", members)
	}
}
```