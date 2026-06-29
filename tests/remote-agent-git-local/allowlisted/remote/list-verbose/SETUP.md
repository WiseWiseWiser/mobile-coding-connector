# Scenario

**Feature**: `git remote -v` lists fetch/push URLs

```
local: remote add origin -> remote-agent remote -v -> origin lines
```

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	gitRun(t, dir, "remote", "add", "origin", "https://example.com/foo.git")
	setGitLocalArgs(t, req, dir, "remote", "-v")
	return nil
}
```