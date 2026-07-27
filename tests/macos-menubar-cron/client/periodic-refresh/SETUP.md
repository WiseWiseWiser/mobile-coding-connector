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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "periodic-refresh"
	return nil
}
```
