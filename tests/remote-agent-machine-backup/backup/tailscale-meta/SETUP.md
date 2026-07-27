# Scenario

**Feature**: mock tailscale running → archive tailscale-config.json + dry-run TAILSCALE section

```
# SeedTailscaleMock -> dry-run then backup -> TAILSCALE section after ENV, archive JSON
```

## Preconditions

`SeedTailscaleMock` writes `serverHome/bin/tailscale` mock CLI and bash/zsh history
with tailscale lines; server subprocess `PATH` prepends `serverHome/bin`.

## Steps

1. `SeedTailscaleMock=true`, `DryRunThenArchive=true`.
2. `OutputPath=tailscale-meta.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/tailscale-meta` (REQUIREMENT-DESIGN-tailscale-config-meta.md).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SeedTailscaleMock = true
	req.DryRunThenArchive = true
	req.OutputPath = "tailscale-meta.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```