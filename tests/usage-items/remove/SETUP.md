# Scenario

**Feature**: deleting a usage item

```
Remove(id) -> dropped item + fallback default + warnings
```

## Preconditions

1. Removing the pinned default reassigns the default to the next enabled item.

## Steps

1. Set `Op=remove` and `TargetID`.

## Context

A silent fallback would leave the menu bar pointing at a missing id, so removal reports it.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "remove"
	return nil
}
```
