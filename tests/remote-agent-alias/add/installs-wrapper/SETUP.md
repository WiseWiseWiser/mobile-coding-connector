# Scenario

**Feature**: `alias add` records the alias and installs its wrapper

```
alias add xdev --server <xdev> -> store entry + ~/.local/bin/xdev-agent
                                -> warning: no token saved yet
```

## Preconditions

Fresh HOME: no alias store, no config, no wrapper.

## Steps

1. `Setup` runs `alias add xdev --server <xdev URL>` with no token seeded.
2. `Assert` checks the store, the wrapper body, its mode, and the warning.

## Context

Primary `add` behavior; no server is contacted.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	setCLI(req, "alias", "add", aliasLeafName, "--server", aliasURLPlaceholder)
	return nil
}
```
