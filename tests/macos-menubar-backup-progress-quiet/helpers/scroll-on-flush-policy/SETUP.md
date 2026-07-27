# Scenario

**Feature**: v1 always scrolls on flush

```
ShouldScrollBackupProgressOnFlush() -> true
```

## Preconditions

Pure bool helper; documents Swift flush behavior (always scroll).

## Steps

1. Op=helper_scroll_policy.

## Context

REQUIREMENT `ShouldScrollBackupProgressOnFlush` for v1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "helper_scroll_policy"
	return nil
}
```
