# Scenario

**Feature**: open still creates and presents the progress window

```
BackupProgressWindow.open
  -> NSWindow(...)
  -> orderFront* / makeKeyAndOrderFront  # some presentation remains
```

## Preconditions

Regression seal: quiet policy must not remove window creation or presentation.

## Steps

1. ClientLeaf=presents-window.

## Context

REQUIREMENT #3; expected GREEN on current code (window + makeKeyAndOrderFront present).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "presents-window"
	return nil
}
```
