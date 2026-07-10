# Scenario

**Feature**: FormatBackupStatusTitle nested Status menu title

```
BackupTaskStatus + now -> "Status: Off" | "Status: On · …"
```

## Preconditions

`Op=status_title`. Sealed strings use middle-dot ` · ` separators.

## Steps

1. Leaf sets Enabled, Phase, LastFinished, NextRun, Now.

## Context

REQUIREMENT status title scenarios 11–13 (+ error preferred).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "status_title"
	if req.NowRFC3339 == "" {
		req.NowRFC3339 = defaultNowRFC3339
	}
	return nil
}
```
