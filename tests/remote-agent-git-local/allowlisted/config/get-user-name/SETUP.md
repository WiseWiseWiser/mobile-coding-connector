# Scenario

**Feature**: `git config --get user.name`

```
gitInitWithMain sets Test User -> config --get user.name
```

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "config", "--get", "user.name")
	return nil
}
```