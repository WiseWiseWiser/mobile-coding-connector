# Scenario

**Feature**: without opencode on PATH, grok and opencode report not installed

```
PATH without opencode -> isAgentInstalled -> grok.installed=false, opencode.installed=false
```

## Preconditions

- PATH reduced to empty bin dir (no opencode).

## Steps

1. `StripOpenCode = true`, `Op = OpListAgents`.

## Context

Mirrors requirement: grok Installed iff opencode resolvable.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpListAgents
	req.StripOpenCode = true
	return nil
}
```