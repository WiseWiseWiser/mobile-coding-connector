# Scenario

**Bug**: BackupProgressWindow steals focus via NSApp.activate

```
# current (bad): open path activates app
BackupProgressWindow.open -> NSApp.activate(ignoringOtherApps: true)

# required: no activate in BackupProgressWindow.swift
BackupProgressWindow.open -> (no NSApp.activate)
```

## Preconditions

`BackupProgressWindow.swift` must not contain `activate(ignoringOtherApps:` or
`NSApp.activate` (or `NSApplication.shared.activate`).

## Steps

1. ClientLeaf=no-activate.

## Context

REQUIREMENT #2; fails on current code that calls `NSApp.activate(ignoringOtherApps: true)`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "no-activate"
	return nil
}
```
