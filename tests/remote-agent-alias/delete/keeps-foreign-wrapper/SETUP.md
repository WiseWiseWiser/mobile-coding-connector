# Scenario

**Feature**: `alias delete` removes the record but never a user-owned wrapper

```
Prelude then user overwrites ~/.local/bin/xdev-agent
  -> alias delete xdev -> record gone, warning, user script kept
```

## Preconditions

`Prelude` installs the alias + wrapper and its token; `MutateWrapper` then
replaces the wrapper with a user script (simulating a hand-edited file).

## Steps

1. `Setup` requests the prelude with `MutateWrapper` set to the user script.
2. `Setup` runs `alias delete xdev`.
3. `Assert` checks the record is gone, the warning is present, and the file
   still holds the user's bytes.

## Context

Deleting must stay safe even after someone repurposes the wrapper path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.Prelude = true
	req.MutateWrapper = aliasForeignBody
	setCLI(req, "alias", "delete", aliasLeafName)
	return nil
}
```
