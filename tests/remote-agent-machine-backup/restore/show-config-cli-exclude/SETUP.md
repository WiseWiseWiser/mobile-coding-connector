# Scenario

**Feature**: restore --show-config without archive merges CLI --exclude into effective preview

```
# no archive argument; same merge path as backup show-config
remote-agent machine restore --show-config --exclude .knowledge-index -> JSON with user excluded
```

## Preconditions

No prereq backup; no archive argument.

## Steps

1. `PrereqBackup=false`, `ShowConfig=true`, `ExcludePaths=[".knowledge-index"]`.
2. Args: `machine restore`.

## Context

REQUIREMENT leaf `restore/show-config-cli-exclude`. Restore preview uses server effective merge, not archive snapshot.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.PrereqBackup = false
	req.ShowConfig = true
	req.ExcludePaths = []string{".knowledge-index"}
	req.Args = []string{"machine", "restore"}
	return nil
}
```