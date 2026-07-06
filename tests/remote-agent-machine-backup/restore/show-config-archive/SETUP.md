# Scenario

**Feature**: restore --show-config with archive prints effective config from archive

```
# prereq backup -> read .backup/config.json from archive
remote-agent machine restore --show-config <archive> -> effective exclude_paths
```

## Preconditions

Prereq backup from default `serverHome` fixtures.

## Steps

1. `ShowConfig=true`.
2. Args: `machine restore` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/show-config-archive`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ShowConfig = true
	req.Args = []string{"machine", "restore"}
	return nil
}
```