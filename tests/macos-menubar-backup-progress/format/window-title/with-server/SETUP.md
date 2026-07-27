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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ServerName = "foo.example.com"
	return nil
}
```
