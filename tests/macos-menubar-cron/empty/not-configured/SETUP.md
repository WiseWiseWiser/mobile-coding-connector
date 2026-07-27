# Scenario

**Feature**: remote Cron menu when endpoint is not configured

```
FormatCronNotConfiguredLabel() -> "Not configured"
```

## Preconditions

Remote app has no configured base URL (same copy as Services/Terminals).

## Steps

1. Set `NotConfigured=true`.

## Context

REQUIREMENT leaf: `empty/not-configured`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.NotConfigured = true
	return nil
}
```
