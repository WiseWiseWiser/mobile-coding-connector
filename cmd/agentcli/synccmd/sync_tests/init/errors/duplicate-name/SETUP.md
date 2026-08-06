# Scenario

**Feature**: init rejects duplicate pair name

```
PreArgs: init mad-max once\noperator -> init mad-max again -> already exists
```

## Preconditions

- PreArgs: successful init for `mad-max`.
- Args: same init again.

## Steps

1. Pre-init pair.
2. Re-init same name; Assert error contains `already exists`.

## Context

- Duplicate name guard.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.FocusPair = "mad-max"
	req.PreArgs = [][]string{
		{"unison", "init", "mad-max", req.LocalPath, req.RemotePath},
	}
	req.Args = []string{"unison", "init", "mad-max", req.LocalPath, req.RemotePath}
	return nil
}
```
