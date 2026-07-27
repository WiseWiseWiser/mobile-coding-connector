# Scenario

**Feature**: local menu bar exposes Projects submenu

```
AICriticApp.swift -> top-level Projects menu present
```

## Preconditions

Local app sources under `macos-ai-critic/ai-critic-macos/`.

## Steps

1. Set `ClientLeaf=projects-submenu`.

## Context

REQUIREMENT scenario 17.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "projects-submenu"
	return nil
}
```
