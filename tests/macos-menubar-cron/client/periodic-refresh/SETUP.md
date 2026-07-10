# Scenario

**Feature**: periodic refresh and Refresh path include cron tasks

```
refresh loop / Button("Refresh") -> listCronTasks / /api/cron-tasks
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=periodic-refresh`.

## Context

REQUIREMENT leaf: `client/periodic-refresh`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "periodic-refresh"
	return nil
}
```
