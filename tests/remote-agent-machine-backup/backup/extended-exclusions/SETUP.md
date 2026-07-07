# Scenario

**Feature**: backup --show-config lists v1.1 built-in exclusions with new rules and paths

```
# local built-in config lookup (no backup API)
remote-agent machine backup --show-config -> JSON version 1.1 + extended exclude_paths
```

## Preconditions

Default `serverHome` fixtures (extended exclusion trees seeded).

## Steps

1. `ShowConfig=true`.
2. Args: `machine backup`.

## Context

REQUIREMENT leaf `backup/extended-exclusions`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ShowConfig = true
	req.Args = []string{"machine", "backup"}
	return nil
}
```