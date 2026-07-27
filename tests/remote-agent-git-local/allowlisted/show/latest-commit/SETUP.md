# Scenario

**Feature**: `git show` includes latest commit subject

```
Initial commit -> show -s --format=%s HEAD -> Initial commit
```

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "show", "-s", "--format=%s", "HEAD")
	return nil
}
```