# Scenario

**Feature**: restore --show-meta prints cloudflared-config.json section

```
# prereq backup with SeedCloudflaredMock -> restore --show-meta -> cloudflared JSON section
```

## Preconditions

Prereq backup from `SeedCloudflaredMock` server home (custom archive).

## Steps

1. `SeedCloudflaredMock=true`, `PrereqBackup=true`, `ShowMeta=true`.
2. Args: `machine restore` (archive injected by Run).

## Context

REQUIREMENT leaf `restore/show-meta-cloudflared`. Complements `restore/show-meta`
(installed.json + ENV only).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedCloudflaredMock = true
	req.PrereqBackup = true
	req.ShowMeta = true
	req.Args = []string{"machine", "restore"}
	return nil
}
```