# Scenario

**Feature**: delete confirmation dialog copy includes task name

```
FormatDeleteCronConfirm("backup") -> `Delete cron task "backup"?`
```

## Preconditions

Confirm dialog uses exact product copy from REQUIREMENT.

## Steps

1. Set `Name=backup`.

## Context

REQUIREMENT: confirm `Delete cron task "{name}"?` before DELETE.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "backup"
	return nil
}
```
