# Scenario

**Feature**: BackupProgressWindow (or family) present for append-only progress

```
remote/Shared Swift -> BackupProgressWindow / open + append lines
```

## Preconditions

AppKit progress/log window for backup job (LogStreamWindow family pattern).

## Steps

1. ClientLeaf=progress-window.

## Context

REQUIREMENT #16.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "progress-window"
	return nil
}
```
