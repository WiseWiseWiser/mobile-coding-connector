# Scenario

**Feature**: no orphan after agent session stop

```
launch grok -> DELETE session -> stopServer uses CleanupOpencodeServe
```

## Preconditions

- Fake opencode serves health endpoint.

## Steps

1. Inherited `ScenarioAgentStop` from parent Setup.

## Context

Regression for Path A orphans when tests only killed parent server process.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.AgentID == "" {
		req.AgentID = "grok"
	}
	return nil
}
```
