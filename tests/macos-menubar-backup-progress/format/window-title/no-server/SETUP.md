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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ServerName = ""
	return nil
}
```
