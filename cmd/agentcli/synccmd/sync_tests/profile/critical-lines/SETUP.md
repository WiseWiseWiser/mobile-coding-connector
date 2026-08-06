# Scenario

**Feature**: profile contains root, ssh, sshargs, servercmd, prefer, ignore

```
operator -> init mad-max\n  -> profile golden-ish critical lines
```

## Preconditions

- FocusPair mad-max after init with defaults.

## Steps

1. Init defaults.
2. Assert profile substrings.

## Context

- Content contract leaf.

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
