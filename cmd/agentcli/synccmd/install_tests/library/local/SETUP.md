# Scenario

**Feature**: Install scope local (only LocalEnsure)

```
operator -> Install(Scope=local, LocalEnsure) -> local side only
```

## Preconditions

- Grouping for local-scope leaves.
- RemoteEnsure must not be invoked on success path.

## Steps

1. Default Scope to `local` when empty.
2. Leaves set success or LocalEnsureErr.

## Context

- Maps to CLI `--local`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Scope == "" {
		req.Scope = "local"
	}
	return nil
}
```
