# Scenario

**Feature**: `edit` saves the staged content when the remote is unchanged

```
# download -> editor -> md5 precondition holds -> POST /api/files/write 200
remote file (or absent) -> remote-agent edit -> Saved/Created + remote updated
```

## Preconditions

Leaf seeds the remote fixture (or leaves it absent), sets `FileArg`, and picks
what the generated editor script writes to the staged copy.

## Steps

1. Leaf seeds `serverHome` via `setEditArgs` / `setEditNewFileArgs` (root SETUP.md).
2. `Run` downloads/stages, runs the fake editor, and posts the write.
3. Assertions expect exit 0, a `Saved`/`Created` line, updated remote bytes,
   and the staged copy kept under the staging dir.

## Context

Happy-path leaves: content change, new file, no change, fresh base download,
editor args, converged content, symlink target, remembered flags.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```
