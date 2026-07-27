# Scenario

**Feature**: `rev-parse HEAD` returns commit hash

```
Initial commit -> rev-parse HEAD -> 40-char hex (+ newline)
```

## Context

REQUIREMENT leaf #6.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "rev-parse", "HEAD")
	return nil
}
```