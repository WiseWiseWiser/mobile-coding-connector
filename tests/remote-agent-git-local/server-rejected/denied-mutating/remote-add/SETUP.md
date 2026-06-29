# Scenario

**Feature**: `git remote add` denied; read-only remote allowed elsewhere

```
remote-agent git -C <repo> remote add origin <url> -> denied
```

## Preconditions

Git repo without remotes.

## Steps

1. `mkDeniedRepo`.
2. Run `remote add origin https://example.com/foo.git`.

## Context

Requirement: remote read-only forms only.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkDeniedRepo(t)
	setGitLocalArgs(t, req, dir, "remote", "add", "origin", "https://example.com/foo.git")
	return nil
}
```