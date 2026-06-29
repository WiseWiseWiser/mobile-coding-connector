# Scenario

**Bug**: `git add` must not run via generic runner

```
remote-agent git -C <repo> add file.txt -> denied
```

## Preconditions

Tracked repo with untracked file.

## Steps

1. Create repo; write `new.txt` without adding.
2. Run `git add new.txt` through remote-agent.

## Context

Mutating subcommand out of scope for `/run`.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkDeniedRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x\n"), 0644); err != nil {
		return err
	}
	setGitLocalArgs(t, req, dir, "add", "new.txt")
	return nil
}
```