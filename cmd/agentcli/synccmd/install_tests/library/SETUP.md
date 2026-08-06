# Scenario

**Feature**: Install library path with injectable ensure hooks

```
operator -> Install(opts{Scope, LocalEnsure, RemoteEnsure})
  -> InstallReport + optional error
```

## Preconditions

- Grouping for library Install leaves.
- Leaves set Scope and Fake* / error fields; harness injects hooks.

## Steps

1. Default Mode to `install` when empty.
2. Leaves configure Scope and hook outcomes.

## Context

- Library-first asserts; CLI default covered under `cli/`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "install"
	}
	return nil
}
```
