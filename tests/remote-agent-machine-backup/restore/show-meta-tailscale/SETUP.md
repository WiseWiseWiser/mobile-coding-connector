# Scenario

**Feature**: restore --show-meta prints tailscale-config.json section

```
# prereq backup with SeedTailscaleMock -> restore --show-meta -> tailscale JSON section
```

## Preconditions

Prereq backup from `SeedTailscaleMock` server home (custom archive).

## Steps

1. `SeedTailscaleMock=true`, `PrereqBackup=true`, `ShowMeta=true`.
2. Args: `machine restore` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/show-meta-tailscale`. Complements `restore/show-meta`
(installed.json + ENV only).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedTailscaleMock = true
	req.PrereqBackup = true
	req.ShowMeta = true
	req.Args = []string{"machine", "restore"}
	return nil
}
```