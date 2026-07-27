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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "menu-placement"
	return nil
}
```
