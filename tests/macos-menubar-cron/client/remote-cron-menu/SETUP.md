# Scenario

**Feature**: remote app exposes Cron menu with accessibility id

```
remote AICriticApp -> Menu("Cron") + accessibilityIdentifier("cron-menu")
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=remote-cron-menu`.

## Context

REQUIREMENT leaf: `client/remote-cron-menu`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "remote-cron-menu"
	return nil
}
```
