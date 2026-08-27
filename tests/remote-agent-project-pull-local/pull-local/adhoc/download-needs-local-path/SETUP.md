# Scenario

**Feature**: download mode requires `--local-path`

```
remote-agent project pull-local --adhoc <dir> --mode download -> error local-path required
```

## Preconditions

Unregistered remote git dir.

## Steps

1. Create remote git repo.
2. Run adhoc download without `--local-path`.

## Context

Download has no existing clone to attach to.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	pair := pairSameOriginRepos(t)
	req.Args = []string{"project", "pull-local", "--adhoc", pair.RemoteDir, "--mode", "download"}
	return nil
}
```
