# Scenario

**Feature**: `git show` includes latest commit subject

```
Initial commit -> show -s --format=%s HEAD -> Initial commit
```

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "show", "-s", "--format=%s", "HEAD")
	return nil
}
```