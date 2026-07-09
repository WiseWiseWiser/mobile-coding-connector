# Scenario

**Feature**: not_configured status guides user to Configure…

```
FormatStatus(not_configured, "") -> guided copy mentioning Configure…
```

## Preconditions

No domains / missing config.

## Steps

1. Set `ConnectionState=not_configured`.

## Context

REQUIREMENT leaf: `status/not-configured`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConnectionState = "not_configured"
	req.StatusServer = ""
	req.Token = "must-not-appear"
	return nil
}
```
