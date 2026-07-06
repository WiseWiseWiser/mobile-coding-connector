# Scenario

**Feature**: backup --show-config prints built-in exclusion config JSON

```
# local built-in config lookup (no backup API)
remote-agent machine backup --show-config -> JSON version + exclude_paths
```

## Preconditions

Default `serverHome` fixtures; server may start but backup API is not required.

## Steps

1. `ShowConfig=true`.
2. Args: `machine backup`.

## Context

REQUIREMENT leaf `backup/show-config`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ShowConfig = true
	req.Args = []string{"machine", "backup"}
	return nil
}
```