# Scenario

**Feature**: headless agent registry via sessionMgr (Path A)

```
sessionMgr.launch -> opencode serve -> opencode-serve-children.json
sessionMgr.stop -> kill + remove entry
```

## Preconditions

- Agent `grok` with fake or real opencode on PATH.

## Steps

1. Set `Op` per leaf (`OpLaunchRegistry`, `OpStopRegistry`, `OpExitRegistry`).

## Context

MECE: write on launch, remove on stop, remove on external exit.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.AgentID = "grok"
	req.UseFakeOpenCode = true
	return nil
}
```
