# Scenario

**Feature**: GET /api/agents exposes Grok in the coding-agent catalog

```
# list handler copies agentDefs and fills installed per agent
GET /api/agents -> Agent catalog -> JSON array
```

## Preconditions

- No full ai-critic server required; handler registered on httptest mux.

## Steps

1. Set `Request.Op = OpListAgents`.
2. Optionally adjust PATH via child Setup for install-mirror leaves.

## Context

Splits on whether we assert catalog fields only or installed parity with opencode.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpListAgents
	return nil
}
```