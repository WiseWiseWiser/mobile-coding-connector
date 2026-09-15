# Scenario

**Feature**: pinning an item stops rotation and is persisted

```
default cc-v2 -> default=cc-v2, rotate=false (written to the registry file)
```

## Preconditions

1. The registry starts seeded (grok + codex, rotating).

## Steps

1. Set `Op=default`, `TargetID=cc-v2`, `Rotate=false`, and register that item.

## Context

`usage default <id>` is how the user chooses which usage the menu bar shows.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "default"
	req.SeedItems = append(grokCodexItems(), commandCodeItems()...)
	req.SeedDefault = "grok"
	req.SeedRotate = true
	req.TargetID = "cc-v2"
	req.Rotate = false
	return nil
}
```
