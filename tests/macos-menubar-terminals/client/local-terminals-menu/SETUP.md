# Scenario

**Feature**: local app exposes Terminals menu

```
ai-critic-macos/AICriticApp.swift -> Menu("Terminals") ...
```

## Preconditions

Local product must list server terminal sessions in a Terminals submenu.

## Steps

1. Set `ClientLeaf=local-terminals-menu`.

## Context

REQUIREMENT leaf: `client/local-terminals-menu`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "local-terminals-menu"
	return nil
}
```
