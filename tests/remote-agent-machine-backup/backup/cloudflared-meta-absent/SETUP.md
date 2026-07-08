# Scenario

**Feature**: no cloudflared mock → omit cloudflared-config.json and CLOUDFLARED dry-run section

```
# default serverHome (no mock) -> dry-run then backup -> no cloudflared meta
```

## Preconditions

Default `serverHome` fixtures without `SeedCloudflaredMock`. Real `cloudflared` absent or not
running in CI.

## Steps

1. `DryRunThenArchive=true` (no `SeedCloudflaredMock`).
2. `OutputPath=cloudflared-meta-absent.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/cloudflared-meta-absent` (REQUIREMENT-DESIGN-cloudflared-config-meta.md).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.DryRunThenArchive = true
	req.OutputPath = "cloudflared-meta-absent.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```