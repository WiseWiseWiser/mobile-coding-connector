# Scenario

**Feature**: `alias add --dry-run` plans the install without writing

```
alias add xdev --server <xdev> --dry-run -> plan lines, no store, no wrapper
```

## Preconditions

Fresh HOME: no alias store, no wrapper.

## Steps

1. `Setup` runs `alias add xdev --server <xdev URL> --dry-run`.
2. `Assert` checks the plan lines and that nothing was created.

## Context

Dry-run is the review path for a state-changing command.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	setCLI(req, "alias", "add", aliasLeafName, "--server", aliasURLPlaceholder, "--dry-run")
	return nil
}
```
