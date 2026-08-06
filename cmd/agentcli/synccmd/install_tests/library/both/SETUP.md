# Scenario

**Feature**: Install scope both (LocalEnsure and RemoteEnsure)

```
operator -> Install(Scope=both|empty) -> both hooks
```

## Preconditions

- Grouping for both-scope leaves.
- Both hooks must be invoked on success.

## Steps

1. Default Scope to `both` when empty.
2. Leaves set Fake versions.

## Context

- Maps to CLI `--both` and default no-flag install.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Scope == "" {
		req.Scope = "both"
	}
	return nil
}
```
