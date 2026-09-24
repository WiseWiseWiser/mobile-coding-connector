# Scenario

**Feature**: dotfiles stage and save like any other file

```
# remote .env -> staged <staging>/<serverHome>/.env -> editor -> Saved
serverHome .env "A=1\n" -> remote-agent edit .env -> .env "A=2\n"
```

## Preconditions

Remote `.env` exists with `A=1\n`.

## Steps

1. Seed `.env` via `setEditArgs`.
2. Editor writes `A=2\n`.
3. Assert exit 0, `Saved`, and updated remote bytes.

## Context

Happy-path leaf #13 — save-success/dotfile. Hidden files are a common edit target
and must not be skipped by staging, browsing, or the write path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, ".env", "A=1\n")
	req.EditorWrite = "A=2\n"
	return nil
}
```
