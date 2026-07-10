# Scenario

**Feature**: text view is non-editable and selectable

```
textView.isEditable = false
textView.isSelectable = true
```

## Preconditions

Both assignments present in `BackupProgressWindow.swift`.

## Steps

1. ClientLeaf=non-editable-selectable.

## Context

REQUIREMENT #11; expected GREEN on current code.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "non-editable-selectable"
	return nil
}
```
