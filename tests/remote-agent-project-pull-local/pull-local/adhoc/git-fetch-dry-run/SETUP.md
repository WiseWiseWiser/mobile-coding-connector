# Scenario

**Feature**: adhoc git-fetch dry-run prints plan without mutating

```
remote-agent project pull-local --adhoc <dir> --mode git-fetch --local-path <local> --dry-run
  -> plan with mode=git-fetch; remote stays dirty
```

## Preconditions

Dirty remote; local same-origin clone; not registered (adhoc).

## Steps

1. Same-origin dirty pair.
2. Dry-run adhoc git-fetch with `--local-path`.

## Context

User-chosen mode; no auto detection.

```go
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir, localDir := pair.RemoteDir, pair.LocalDir
	// Dirty the remote.
	p := filepath.Join(remoteDir, "adhoc-dirty.txt")
	if err := os.WriteFile(p, []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	req.Args = []string{
		"project", "pull-local",
		"--adhoc", remoteDir,
		"--mode", "git-fetch",
		"--local-path", localDir,
		"--dry-run",
	}
	req.LocalPath = localDir
	req.RemoteDirAfterSetup = remoteDir
	return nil
}
```
