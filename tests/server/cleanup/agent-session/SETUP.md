# Scenario

**Feature**: explicit agent session stop cleans child port

```
POST /api/agents/sessions -> DELETE stop -> stopServer -> no listener on session port
```

## Preconditions

- Server running with launched grok session.

## Steps

1. `Scenario = ScenarioAgentStop`.

## Context

Validates stop API + harness cleanup helper together.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Scenario = ScenarioAgentStop
	return nil
}
```
