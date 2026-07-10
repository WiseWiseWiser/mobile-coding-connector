# Scenario

**Feature**: Cron menu sits after Services and before Terminals

```
Menu("Services") ... Menu("Cron") ... Menu("Terminals") in source order
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=menu-placement`.

## Context

REQUIREMENT leaf: `client/menu-placement`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "menu-placement"
	return nil
}
```
