# Scenario

**Feature**: local menu bar exposes New Cron Task…

```
local AICriticApp Menu("Cron") -> Button("New Cron Task…")
```

## Preconditions

Local Swift app sources present under `macos-ai-critic/ai-critic-macos/`.

## Steps

1. Set `ClientLeaf=local-new-cron-task`.

## Context

REQUIREMENT leaf: `client/local-new-cron-task` (scenario 4).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "local-new-cron-task"
	return nil
}
```
