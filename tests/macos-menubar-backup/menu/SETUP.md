# Scenario

**Feature**: Status nested Enable/Disable children and item enablement

```
BackupStatusMenuChildren() -> ["Enable","Disable"]
BackupEnableItemEnabled / BackupDisableItemEnabled(taskEnabled)
```

## Preconditions

Status submenu has **only** Enable and Disable (not Backup Now, which is sibling).

## Steps

1. Leaf sets `Op` to `menu_children` or `menu_gating`.

## Context

REQUIREMENT menu scenarios 21–23.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Grouping: leaves assign Op (menu_children | menu_gating).
	if req == nil {
		t.Fatal("req is nil")
	}
	return nil
}
```
