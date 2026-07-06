# Scenario

**Feature**: restore --show-meta prints archive meta except config.json

```
# prereq backup -> read .backup/installed.json and .backup/ENV
remote-agent machine restore --show-meta <archive> -> sectioned stdout
```

## Preconditions

Prereq backup from default `serverHome` fixtures.

## Steps

1. `ShowMeta=true`.
2. Args: `machine restore` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/show-meta`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ShowMeta = true
	req.Args = []string{"machine", "restore"}
	return nil
}
```