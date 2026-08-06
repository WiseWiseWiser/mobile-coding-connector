# Scenario

**Feature**: Install scope remote (only RemoteEnsure)

```
operator -> Install(Scope=remote, RemoteEnsure(targetPath)) -> remote side only
```

## Preconditions

- Grouping for remote-scope leaves.
- LocalEnsure must not be invoked on success path.
- RemoteTargetPath from root Setup.

## Steps

1. Default Scope to `remote` when empty.
2. Leaves set FakeRemoteVersion.

## Context

- Maps to CLI `--remote`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Scope == "" {
		req.Scope = "remote"
	}
	return nil
}
```
