# Scenario

**Feature**: mock cloudflared running → archive cloudflared-config.json + dry-run CLOUDFLARED section

```
# SeedCloudflaredMock -> dry-run then backup -> CLOUDFLARED section after ENV, archive JSON
```

## Preconditions

`SeedCloudflaredMock` writes `serverHome/bin/cloudflared` mock CLI, `serverHome/bin/pgrep`
stub (reads `.doctest-cloudflared.pid`), `.doctest-cloudflared.pid` + `.doctest-cloudflared.cmdline`,
`.cloudflared/config.yml` with fake credentials, bash history with cloudflared quick-tunnel line;
server subprocess `PATH` prepends `serverHome/bin`.

## Steps

1. `SeedCloudflaredMock=true`, `DryRunThenArchive=true`.
2. `OutputPath=cloudflared-meta.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/cloudflared-meta` (REQUIREMENT-DESIGN-cloudflared-config-meta.md).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedCloudflaredMock = true
	req.DryRunThenArchive = true
	req.OutputPath = "cloudflared-meta.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```