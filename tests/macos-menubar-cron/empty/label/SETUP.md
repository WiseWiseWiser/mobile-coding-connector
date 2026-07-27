# Scenario

**Feature**: empty cron task list label

```
FormatCronTasksEmptyLabel() -> "No cron tasks configured"
```

## Preconditions

Cron list is empty but server/endpoint is available.

## Steps

1. Leave `NotConfigured=false` (default).

## Context

REQUIREMENT leaf: `empty/label`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NotConfigured = false
	return nil
}
```
