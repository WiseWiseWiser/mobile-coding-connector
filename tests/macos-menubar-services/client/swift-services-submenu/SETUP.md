# Scenario

**Feature**: nested per-service SwiftUI Menu

```
ForEach(services) -> Menu(title) -> Start | Restart | Stop | Disable/Enable | View Logs
```

## Preconditions

`AICriticApp.swift` renders Services submenu with nested `Menu` per service.

## Steps

1. Set `ClientLeaf=swift-services-submenu`.

## Context

REQUIREMENT leaf: `client/swift-services-submenu`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "swift-services-submenu"
	return nil
}
```