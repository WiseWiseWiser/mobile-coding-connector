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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.NotConfigured = true
	return nil
}
```
