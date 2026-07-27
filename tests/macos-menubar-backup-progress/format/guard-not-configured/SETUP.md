# Scenario

**Feature**: guard failure — not configured (visible in window)

```
FormatBackupProgressGuardError("not_configured") -> "ERROR: not configured"
```

## Preconditions

Early failure before network; must not be a silent return only.

## Steps

1. Op=format_guard; GuardReason=not_configured.

## Context

REQUIREMENT goal 5; guard table.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format_guard"
	req.GuardReason = "not_configured"
	return nil
}
```
