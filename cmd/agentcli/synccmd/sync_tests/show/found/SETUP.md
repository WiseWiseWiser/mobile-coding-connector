# Scenario

**Feature**: show prints fields for an existing pair

```
PreArgs: init mad-max\noperator -> show mad-max -> stdout has name/local/remote
```

## Preconditions

- Pre-init mad-max.

## Steps

1. PreArgs init.
2. Args show mad-max.
3. Assert key fields appear on stdout.

## Context

- Found path.

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
	req.Args = []string{"unison", "show", "mad-max"}
	return nil
}
```
