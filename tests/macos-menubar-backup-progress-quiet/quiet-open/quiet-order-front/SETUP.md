# Scenario

**Feature**: open uses non-activating order front

```
# required quiet presentation
BackupProgressWindow.open
  -> window.orderFrontRegardless()  # or orderFront( without makeKey)
  # not only makeKeyAndOrderFront + activate
```

## Preconditions

Source must show `orderFrontRegardless` or bare `orderFront(` after stripping
`makeKeyAndOrderFront` (which steals key and is insufficient alone).

## Steps

1. ClientLeaf=quiet-order-front.

## Context

REQUIREMENT #1; current code only uses `makeKeyAndOrderFront` → RED until quiet front.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "quiet-order-front"
	return nil
}
```
