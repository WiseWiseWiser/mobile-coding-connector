# Scenario

**Feature**: grok agent integration doctest harness

```
# catalog lists grok; installed mirrors opencode binary resolution
GET /api/agents -> Agent catalog -> JSON with grok + opencode.installed

# headless launch reuses opencode serve on project dir
POST launch grok -> Session manager -> opencode serve -> /global/health

# model preference uses agent-specific substring after ready
applyPreferredModel -> PATCH /config with grok* model when agent_id=grok
```

## Preconditions

- Tests call `agents.TestExported_*` helpers (implementer provides in `export_test.go`).
- Fake opencode binary built from `testdata/fake-opencode` when `UseFakeOpenCode` is set.
- `AI_CRITIC_HOME` may use isolated config; agent list does not require a running server.

## Steps

1. Child `Setup` sets `Request.Op` and scenario fields (`StripOpenCode`, `UseFakeOpenCode`, etc.).
2. Root `Run` adjusts PATH, builds fake opencode if needed, invokes list or launch.
3. Leaf `Assert` validates `Response`.

## Context

Covers requirement REQUIREMENT-DESIGN-integrate-grok-agent: five agents in list, grok
headless launch, install parity with opencode, grok model substring preference.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.AgentID == "" && req.Op == OpLaunchGrok {
		req.AgentID = "grok"
	}
	return nil
}
```