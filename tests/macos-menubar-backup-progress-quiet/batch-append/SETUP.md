# Scenario

**Feature**: ProgressSession batches lines and flushes cheaply on an interval

```
# low-CPU append path
ProgressSession.append(line)
  -> pendingLines / buffer
  -> flush on ~100–200ms timer
  -> textStorage.append(batch)
  -> scrollToEnd once per flush
  # not per-line textView.string += + scroll
```

## Preconditions

All contracts inspect `BackupProgressWindow.swift` ProgressSession append/flush path.

## Steps

1. Set `Op=client`; leaf sets `ClientLeaf`.

## Context

REQUIREMENT low-CPU append scenarios 4–7.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
