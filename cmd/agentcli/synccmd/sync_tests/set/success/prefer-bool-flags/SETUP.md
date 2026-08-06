# Scenario

**Feature**: set prefer and boolean flags

```
PreArgs: init\noperator -> set --prefer older --no-times --no-auto --no-batch\n  -> fields false/older
```

## Preconditions

- Pair mad-max after init with defaults true.

## Steps

1. Pre-init.
2. Set prefer + no-times/auto/batch.
3. Assert field values.

## Context

- Bool and prefer patch.

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
		"--prefer", "older",
		"--no-times", "--no-auto", "--no-batch",
	}
	return nil
}
```
