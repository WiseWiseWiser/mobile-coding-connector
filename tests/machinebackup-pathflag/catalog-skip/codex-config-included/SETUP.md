# Scenario

**Feature**: ordinary Codex config is not catalog-excluded

```
MergeExclusions(nil,nil,nil) -> IsExcluded(".codex/config.toml") == false
# owner-only / no DefaultSkipMask flags
```

## Preconditions

- No path catalog rule covers `.codex/config.toml` (owner may still be codex).

## Steps

1. RelPath `.codex/config.toml`.
2. Expect not excluded.

## Context

- Negative control next to `.codex/.tmp` skip.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.RelPath = ".codex/config.toml"
	req.WantExcluded = false
	req.WantExcludedSet = true
	return nil
}
```
