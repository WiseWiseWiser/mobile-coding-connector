# Scenario

**Feature**: set --ignore replaces the ignore list

```
PreArgs: init (default ignores)\noperator -> set --ignore 'Name foo' --ignore 'Path bar'\n  -> ignore is exactly those two; profile ignore lines updated
```

## Preconditions

- Replace semantics when any `--ignore` is passed.

## Steps

1. Pre-init.
2. Set two ignore lines.
3. Assert ignore slice equals the two entries (not merged with defaults).

## Context

- Ignore replace.

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
	req.Args = []string{
		"unison", "set", "mad-max",
		"--ignore", "Name foo",
		"--ignore", "Path bar",
	}
	return nil
}
```
