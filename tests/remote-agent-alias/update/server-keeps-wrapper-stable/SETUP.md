# Scenario

**Feature**: `alias update --server` retargets the alias without touching the wrapper

```
Prelude -> alias update xdev --server https://stage.example.com
        -> store server updated; wrapper bytes identical
```

## Preconditions

`Prelude` installs the alias and its wrapper, and saves a token for the old server.

## Steps

1. `Setup` requests the prelude and runs `alias update xdev --server <new URL>`.
2. `Assert` compares the wrapper before/after and reads the store.

## Context

The wrapper carries only `--alias`, which is why retargeting needs no rewrite.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

const aliasStageServer = "https://stage.example.com"

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.Prelude = true
	setCLI(req, "alias", "update", aliasLeafName, "--server", aliasStageServer)
	return nil
}
```
