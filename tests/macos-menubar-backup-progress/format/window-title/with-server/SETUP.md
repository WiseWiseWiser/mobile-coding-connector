# Scenario

**Feature**: window title includes server name

```
FormatBackupProgressWindowTitle("foo.example.com") -> "Backup: foo.example.com"
```

## Preconditions

Non-empty server scope.

## Steps

1. ServerName=foo.example.com.

## Context

REQUIREMENT Window title.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ServerName = "foo.example.com"
	return nil
}
```
