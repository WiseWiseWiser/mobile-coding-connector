# Scenario

**Feature**: launch persists opencode-serve-children.json entry

```
TestExported_LaunchAgentSession(grok) -> registry entry with pid/port/session_id
```

## Preconditions

- Fake opencode serves /global/health (fast). Real opencode optional via `UseRealOpenCode`.

## Steps

1. `Op = OpLaunchRegistry`, `UseFakeOpenCode = true`.

## Context

Core Path A registration immediately after `cmd.Start()`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLaunchRegistry
	req.UseFakeOpenCode = true
	return nil
}
```
