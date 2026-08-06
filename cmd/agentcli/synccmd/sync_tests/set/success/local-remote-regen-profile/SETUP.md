# Scenario

**Feature**: set local/remote updates store and regenerates profile roots

```
PreArgs: init mad-max with L0/R0
operator -> set --local L1 --remote R1
  -> pair.local/remote updated; profile roots reflect L1 and ssh://remote-agent//R1
```

## Preconditions

- Pair `mad-max` created with initial workspace paths.
- New local/remote are absolute dirs under `d.DOCTEST_CASE` (`workspace-local-v2`, `workspace-remote-v2`).

## Steps

1. PreArgs: init with original LocalPath/RemotePath from root Setup.
2. Args: set name with `--local` / `--remote` pointing at v2 dirs.
3. Assert store fields and profile root lines.

## Context

- Profile regen is part of CLI set.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	initLocal := req.LocalPath
	initRemote := req.RemotePath
	newLocal := filepath.Join(d.DOCTEST_CASE, "workspace-local-v2")
	newRemote := filepath.Join(d.DOCTEST_CASE, "workspace-remote-v2")
	req.FocusPair = "mad-max"
	req.PreArgs = [][]string{
		{"unison", "init", "mad-max", initLocal, initRemote},
	}
	req.Args = []string{
		"unison", "set", "mad-max",
		"--local", newLocal,
		"--remote", newRemote,
	}
	// So Run mkdirs the new paths:
	req.LocalPath = newLocal
	req.RemotePath = newRemote
	return nil
}
```
