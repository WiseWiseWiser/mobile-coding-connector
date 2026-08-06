# Scenario

**Feature**: add is an alias of init

```
operator -> RunCLI([unison add demo <local> <remote>]) -> pair demo created
```

## Preconditions

- Verb: `add` (not init).
- FocusPair: `demo`.

## Steps

1. Args use `add`.
2. Assert pair exists with defaults.

## Context

- Alias coverage.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.FocusPair = "demo"
	req.Args = []string{"unison", "add", "demo", req.LocalPath, req.RemotePath}
	return nil
}
```
