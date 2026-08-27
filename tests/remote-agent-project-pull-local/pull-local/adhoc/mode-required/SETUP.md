# Scenario

**Feature**: adhoc pull-local requires `--mode`

```
remote-agent project pull-local --adhoc <abs-dir> -> error --mode required
```

## Preconditions

Unregistered remote git dir (not added to projects.json).

## Steps

1. Create a temp remote git repo with a commit.
2. Run pull-local `--adhoc <dir>` without `--mode`.

## Context

`--mode` is mandatory for adhoc; no auto strategy.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	pair := pairSameOriginRepos(t)
	remoteDir := pair.RemoteDir
	// Do NOT register as a project — force adhoc path.
	req.Args = []string{"project", "pull-local", "--adhoc", remoteDir}
	return nil
}
```
