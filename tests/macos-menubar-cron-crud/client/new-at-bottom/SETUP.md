# Scenario

**Feature**: New Cron Task… is at the bottom of the Cron menu

```
Menu("Cron") { tasks…; Divider(); New Cron Task… }  // New last
```

## Preconditions

Empty list still shows empty label + divider + New Cron Task… (New remains
bottom item).

## Steps

1. Set `ClientLeaf=new-at-bottom`.

## Context

REQUIREMENT leaf: `client/new-at-bottom` (scenario 4, placement).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "new-at-bottom"
	return nil
}
```
