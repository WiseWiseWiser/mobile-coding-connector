# Scenario

**Feature**: success footer line

```
FormatBackupProgressStatusSuccess() -> "Status: Success"
```

## Preconditions

Terminal success after write.

## Steps

1. Op=format_status_success.

## Context

REQUIREMENT #13.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "format_status_success"
	return nil
}
```
