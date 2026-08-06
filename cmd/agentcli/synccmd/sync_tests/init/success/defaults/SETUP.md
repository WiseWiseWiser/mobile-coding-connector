# Scenario

**Feature**: init with defaults writes store and profile

```
operator -> RunCLI([unison init mad-max <local> <remote>])\n  -> pairs.json + remote-agent-mad-max.prf with defaults
```

## Preconditions

- Name: `mad-max`.
- Local/Remote: abs paths from case workspace dirs.
- FocusPair: `mad-max`.

## Steps

1. Set Args to init with three positionals.
2. Assert pair defaults and profile exists.

## Context

- Default field matrix from DSN.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.FocusPair = "mad-max"
	req.Args = []string{"unison", "init", "mad-max", req.LocalPath, req.RemotePath}
	return nil
}
```
