# Scenario

**Feature**: window title placeholder when no server

```
FormatBackupProgressWindowTitle("") -> "Backup: (no server)"
```

## Preconditions

Empty server name (guard path still opens window with placeholder title).

## Steps

1. ServerName empty.

## Context

REQUIREMENT Window title `(no server)`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ServerName = ""
	return nil
}
```
