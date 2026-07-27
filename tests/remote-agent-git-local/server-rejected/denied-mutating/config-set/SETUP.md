# Scenario

**Feature**: `git config --set` denied; read-only config allowed elsewhere

```
remote-agent git -C <repo> config --set core.commentChar % -> denied
```

## Preconditions

Git repo.

## Steps

1. `mkDeniedRepo`.
2. Run `config --set core.commentChar %`.

## Context

Requirement allowlist: config read-only only.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	dir := mkDeniedRepo(t)
	setGitLocalArgs(t, req, dir, "config", "--set", "core.commentChar", "%")
	return nil
}
```