# Scenario

**Feature**: empty stash list succeeds

```
clean repo -> stash list -> empty stdout, exit 0
```

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "stash", "list")
	return nil
}
```