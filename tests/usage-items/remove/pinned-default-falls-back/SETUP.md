# Scenario

**Feature**: removing the pinned default falls back to the next enabled item

```
remove cc-v1 (pinned) -> removed cc-v1, default grok, warning
```

## Preconditions

1. `cc-v1` is the pinned default and rotation is off.

## Steps

1. Set `Op=remove` with `TargetID=cc-v1`.

## Context

A silent fallback would leave the menu bar showing a missing id.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "remove"
	req.SeedItems = append(grokCodexItems(), commandCodeItems()...)
	req.SeedDefault = "cc-v1"
	req.SeedRotate = false
	req.TargetID = "cc-v1"
	return nil
}
```
