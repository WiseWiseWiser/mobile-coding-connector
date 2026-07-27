# Scenario

**Feature**: remote app exposes Terminals menu

```
ai-critic-remote-macos/AICriticApp.swift -> Menu("Terminals") ...
```

## Preconditions

Remote product must list server terminal sessions in a Terminals submenu.

## Steps

1. Set `ClientLeaf=remote-terminals-menu`.

## Context

REQUIREMENT leaf: `client/remote-terminals-menu`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "remote-terminals-menu"
	return nil
}
```
