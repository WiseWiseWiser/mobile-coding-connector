# Scenario

**Feature**: progress UI remains a scrollable monospaced console

```
BackupProgressWindow.open
  -> NSScrollView
  -> NSTextView as documentView
  -> isEditable=false, isSelectable=true
```

## Preconditions

Regression seals for layout; should stay GREEN across quiet/batch refactors.

## Steps

1. Set `Op=client`; leaf sets `ClientLeaf`.

## Context

REQUIREMENT scrollable console scenarios 10–11.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
