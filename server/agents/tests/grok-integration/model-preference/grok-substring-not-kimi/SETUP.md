# Scenario

**Feature**: grok agent uses grok model substring preference

```
PreferredModelSubstringForAgent("grok") -> "grok" (not kimi-k2.5)
```

## Preconditions

- Implementer adds per-agent preference map or branch for AgentID grok.

## Steps

1. `Op = OpModelSubstring`.

## Context

Default opencode preference remains kimi-k2.5; grok is distinct.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpModelSubstring
	return nil
}
```