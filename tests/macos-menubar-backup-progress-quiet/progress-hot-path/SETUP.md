# Scenario

**Feature**: onProgress only enqueues; UI flush is separate inside ProgressSession

```
# network / download callback
MachineBackupClient.onProgress -> format line
  -> progressSession?.append(line)   # enqueue only
ProgressSession.flush (timer) -> textStorage + scroll
```

## Preconditions

Inspects `AICriticApp.swift`, `MachineBackupClient.swift`, and
`BackupProgressWindow.swift` together.

## Steps

1. Set `Op=client`; leaf sets `ClientLeaf`.

## Context

REQUIREMENT progress hot path scenario 8.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
