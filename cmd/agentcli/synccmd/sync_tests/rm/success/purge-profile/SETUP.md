# Scenario

**Feature**: rm removes pair and purges profile by default

```
PreArgs: init mad-max (creates prf)\noperator -> rm mad-max [--yes] [--purge-profile]\n  -> pair gone; profile file removed
```

## Preconditions

- Default purge behavior: profile deleted.
- Use explicit `--purge-profile` and `--yes` for clarity.

## Steps

1. Pre-init.
2. Rm with purge.
3. Assert pair absent and profile missing.

## Context

- Purge path.

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
	req.Args = []string{"unison", "rm", "mad-max", "--yes", "--purge-profile"}
	return nil
}
```
