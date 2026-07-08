# Scenario

**Feature**: mock systemctl with zero running units → archive JSON + dry-run (0 running)

```
# SeedSystemdMock + SeedSystemdMockEmpty -> dry-run then backup -> (0 running) section
```

## Preconditions

`SeedSystemdMock` writes mock `systemctl`; `SeedSystemdMockEmpty` sets `SYSTEMD_MOCK_EMPTY=1`
so list-units returns empty JSON arrays for both scopes.

## Steps

1. `SeedSystemdMock=true`, `SeedSystemdMockEmpty=true`, `DryRunThenArchive=true`.
2. `OutputPath=systemd-services-meta-empty.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/systemd-services-meta-empty` (REQUIREMENT-DESIGN-systemd-services-meta.md).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedSystemdMock = true
	req.SeedSystemdMockEmpty = true
	req.DryRunThenArchive = true
	req.OutputPath = "systemd-services-meta-empty.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```