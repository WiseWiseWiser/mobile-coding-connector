# Scenario

**Feature**: empty recent list placeholder label

```
FormatBackupRecentEmptyLabel() -> "No recent backups"
```

## Preconditions

No archives under the active server backup dir (or empty entry list).

## Steps

1. Set `Op=recent_empty`.

## Context

REQUIREMENT #14.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "recent_empty"
	return nil
}
```
