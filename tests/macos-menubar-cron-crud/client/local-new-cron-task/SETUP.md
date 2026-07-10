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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "local-new-cron-task"
	return nil
}
```
