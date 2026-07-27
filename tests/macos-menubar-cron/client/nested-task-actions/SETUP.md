# Scenario

**Feature**: nested per-task Cron submenu with core actions

```
ForEach cron task -> Menu(title) { Run Now; Enable|Disable; View Logs; History disabled }
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=nested-task-actions`.

## Context

REQUIREMENT leaf: `client/nested-task-actions`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "nested-task-actions"
	return nil
}
```
