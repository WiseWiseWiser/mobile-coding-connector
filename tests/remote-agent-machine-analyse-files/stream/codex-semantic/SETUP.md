# Scenario

**Feature**: `.codex` entry shows children before semantic enricher lines

```
# .codex tree with sessions rollouts + skills dir -> semantic counts in entry block
.codex block: > sessions before sessions N rollouts; summary includes codex lines
```

## Preconditions

`SeedProfile=codex`: two `rollout-*.jsonl` under `sessions/**`, one top-level `skills/` dir,
plus `plain-dir` for contrast.

## Steps

1. Set `SeedProfile` to `codex`.

## Context

REQUIREMENT leaf `stream/codex-semantic`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedProfile = "codex"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```