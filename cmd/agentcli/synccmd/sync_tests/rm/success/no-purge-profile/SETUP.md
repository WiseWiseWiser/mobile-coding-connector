# Scenario

**Feature**: rm --no-purge-profile keeps the .prf file

```
PreArgs: init\noperator -> rm --no-purge-profile\n  -> pair gone; profile file remains
```

## Preconditions

- Explicit `--no-purge-profile`.

## Steps

1. Pre-init.
2. Rm with no-purge.
3. Assert pair gone, profile still exists.

## Context

- Keep profile path.

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
	req.Args = []string{"unison", "rm", "mad-max", "--yes", "--no-purge-profile"}
	return nil
}
```
