# Scenario

**Feature**: periodic services + terminals refresh interval constant

```
menubar.PeriodicRefreshInterval -> 30s background poll
```

## Preconditions

`Op=interval` reads `menubar.PeriodicRefreshInterval`.

## Steps

1. Leaf invokes interval op (no extra inputs).

## Context

REQUIREMENT: app-side poll e.g. ~30s for services + terminals.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "interval"
	return nil
}
```
