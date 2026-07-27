# Scenario

**Feature**: `git config --get user.name`

```
gitInitWithMain sets Test User -> config --get user.name
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
	setGitLocalArgs(t, req, dir, "config", "--get", "user.name")
	return nil
}
```