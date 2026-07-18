# Scenario

**Feature**: BuiltinExclusionConfig is the pathflag catalog SSoT surface

```
BuiltinExclusionConfig() -> exclude_paths
  contains every pathflag catalog rule + synthetic **(binary)
  reasons match pathflag Classify reasons for shared path rules
```

## Preconditions

- Product still lists `**(binary)` even though Classify has no binary content rule.
- Implementer may generate config from pathflag attributeRules + specials.

## Steps

1. Group marks SSoT ops (leaves set Op).
2. Leaves check path set and reason map.

## Context

- May be GREEN if dual tables stay hand-synced; RED-forcing is module/import + log API.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// SSoT leaves inspect BuiltinExclusionConfig only.
	req.RelPath = ""
	return nil
}
```
