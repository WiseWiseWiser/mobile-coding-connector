# Scenario

**Feature**: BackupProgressWindow opens without stealing focus

```
# quiet open policy
BackupProgressWindow.open / openBackupProgress
  -> NSWindow
  -> orderFrontRegardless | quiet orderFront
  # no NSApp.activate(ignoringOtherApps:)
```

## Preconditions

Contracts inspect `BackupProgressWindow.swift` only for activate/order-front tokens.

## Steps

1. Set `Op=client`; leaf sets `ClientLeaf`.

## Context

REQUIREMENT quiet open scenarios 1–3.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
