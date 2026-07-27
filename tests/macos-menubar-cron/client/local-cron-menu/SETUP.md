# Scenario

**Feature**: local app exposes Cron menu with accessibility id

```
local AICriticApp -> Menu("Cron") + accessibilityIdentifier("cron-menu")
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=local-cron-menu`.

## Context

REQUIREMENT leaf: `client/local-cron-menu`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "local-cron-menu"
	return nil
}
```
