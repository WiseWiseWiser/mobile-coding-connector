# Scenario

**Feature**: `alias add` never clobbers a user script at the wrapper path

```
user script at ~/.local/bin/xdev-agent -> alias add xdev -> Error: … pass --force
                                       -> script unchanged, no store entry
```

## Preconditions

`SeedForeignWrapper` pre-creates a `#!/bin/sh` user script at the wrapper path.

## Steps

1. `Setup` runs `alias add xdev --server <xdev URL>` without `--force`.
2. `Assert` checks the error, the untouched script, and the absent record.

## Context

Ownership is detected by the generated marker; this is the guard rail that keeps
`alias` from deleting or overwriting unrelated files.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.SeedForeignWrapper = true
	setCLI(req, "alias", "add", aliasLeafName, "--server", aliasURLPlaceholder)
	return nil
}
```
