# Scenario

**Feature**: no tailscale mock → omit tailscale-config.json and TAILSCALE dry-run section

```
# default serverHome (no mock) -> dry-run then backup -> no tailscale meta
```

## Preconditions

Default `serverHome` fixtures without `SeedTailscaleMock`. Real `tailscale` absent or not
running in CI.

## Steps

1. `DryRunThenArchive=true` (no `SeedTailscaleMock`).
2. `OutputPath=tailscale-meta-absent.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/tailscale-meta-absent` (REQUIREMENT-DESIGN-tailscale-config-meta.md).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.DryRunThenArchive = true
	req.OutputPath = "tailscale-meta-absent.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```