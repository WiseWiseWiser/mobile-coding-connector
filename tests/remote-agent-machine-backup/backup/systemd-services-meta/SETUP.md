# Scenario

**Feature**: mock systemctl available → archive systemd-services.json + dry-run SYSTEMD section

```
# SeedSystemdMock -> dry-run then backup -> SYSTEMD SERVICES section after ENV, archive JSON
```

## Preconditions

`SeedSystemdMock` writes `serverHome/bin/systemctl` mock CLI; server subprocess `PATH`
prepends `serverHome/bin`. Mock returns 1 user + 2 system running service units.

## Steps

1. `SeedSystemdMock=true`, `DryRunThenArchive=true`.
2. `OutputPath=systemd-services-meta.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/systemd-services-meta` (REQUIREMENT-DESIGN-systemd-services-meta.md).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedSystemdMock = true
	req.DryRunThenArchive = true
	req.OutputPath = "systemd-services-meta.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```