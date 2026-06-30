# Scenario

**Feature**: agent catalog includes Grok entry with headless opencode command

```
GET /api/agents -> Agent catalog -> entry id=grok, command=opencode, headless=true
```

## Preconditions

- Default PATH (real opencode may or may not be present; this leaf does not assert installed).

## Steps

1. `Request.Op = OpListAgents`.

## Context

Validates static agent definition fields for Grok.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpListAgents
	return nil
}
```