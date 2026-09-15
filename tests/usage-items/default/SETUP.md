# Scenario

**Feature**: choosing what the menu bar shows

```
SetDefault(id, rotate=false) -> pinned item
SetDefault(id, rotate=true)  -> rotate over enabled items in registry order
```

## Preconditions

1. `default <id>` turns rotation off; `--rotate` turns it back on.

## Steps

1. Set `Op=default`, `TargetID`, and `Rotate`.

## Context

The registry `default` field is the single source of truth for the menu-bar title.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "default"
	return nil
}
```
