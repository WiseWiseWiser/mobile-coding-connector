# Scenario

**Feature**: remote menu bar exposes New Cron Task…

```
remote AICriticApp Menu("Cron") -> Button("New Cron Task…")
  // disabled when remote not configured (source may gate with config flag)
```

## Preconditions

Remote Swift app sources present under `macos-ai-critic/ai-critic-remote-macos/`.

## Steps

1. Set `ClientLeaf=remote-new-cron-task`.

## Context

REQUIREMENT leaf: `client/remote-new-cron-task` (scenario 4).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "remote-new-cron-task"
	return nil
}
```
