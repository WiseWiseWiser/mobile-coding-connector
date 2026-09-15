# Scenario

**Feature**: the two Command Code sandboxes render distinct menu text

```
cc-v1 + cc-v2 -> titles and dropdown lines the app prints verbatim
```

## Preconditions

1. The registry holds grok, codex, and both Command Code items, pinned to cc-v1.

## Steps

1. Set `Op=list` with `SeedItems`, `SeedDefault=cc-v1`, and `SeedRotate=false`.

## Context

This is the shape the user asked for: two Command Code usages side by side.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "list"
	req.SeedItems = append(grokCodexItems(), commandCodeItems()...)
	req.SeedDefault = "cc-v1"
	req.SeedRotate = false
	return nil
}
```
