# Scenario

**Feature**: summary omits tool lines when indicator dir absent

```
# .codex present, .grok absent
summary shows codex lines but not grok sessions/projects/skills
```

## Preconditions

`SeedProfile=topic-absent`: seeds `.codex` only (no `.grok` directory).

## Steps

1. Set `SeedProfile` to `topic-absent`.

## Context

REQUIREMENT leaf `stream/topic-absent`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedProfile = "topic-absent"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```