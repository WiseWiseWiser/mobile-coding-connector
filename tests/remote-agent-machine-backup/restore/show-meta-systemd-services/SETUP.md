# Scenario

**Feature**: restore --show-meta prints systemd-services.json section

```
# prereq backup with SeedSystemdMock -> restore --show-meta -> systemd JSON section
```

## Preconditions

Prereq backup from `SeedSystemdMock` server home (custom archive).

## Steps

1. `SeedSystemdMock=true`, `PrereqBackup=true`, `ShowMeta=true`.
2. Args: `machine restore` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/show-meta-systemd-services`. Complements `restore/show-meta`
(installed.json + ENV only).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedSystemdMock = true
	req.PrereqBackup = true
	req.ShowMeta = true
	req.Args = []string{"machine", "restore"}
	return nil
}
```