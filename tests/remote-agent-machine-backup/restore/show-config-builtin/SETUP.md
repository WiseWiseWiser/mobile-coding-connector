# Scenario

**Feature**: restore --show-config without archive prints built-in config

```
# local built-in config lookup (no archive, no restore API)
remote-agent machine restore --show-config -> JSON version + exclude_paths
```

## Preconditions

No prereq backup; no archive argument.

## Steps

1. `PrereqBackup=false`, `ShowConfig=true`.
2. Args: `machine restore`.

## Context

REQUIREMENT leaf `restore/show-config-builtin`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.PrereqBackup = false
	req.ShowConfig = true
	req.Args = []string{"machine", "restore"}
	return nil
}
```