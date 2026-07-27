# Scenario

**Feature**: failure footer line

```
FormatBackupProgressStatusFailed() -> "Status: Failed"
```

## Preconditions

Terminal failure after ERROR line (or guard).

## Steps

1. Op=format_status_failed.

## Context

REQUIREMENT #11 failure end.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format_status_failed"
	return nil
}
```
