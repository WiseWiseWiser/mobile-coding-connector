# Scenario

**Feature**: no systemctl mock → omit systemd-services.json and SYSTEMD dry-run section

```
# default serverHome (no mock) -> dry-run then backup -> no systemd meta
```

## Preconditions

Default `serverHome` fixtures without `SeedSystemdMock`. Real `systemctl` absent or not
available in CI (macOS/launchd out of scope v1).

## Steps

1. `DryRunThenArchive=true` (no `SeedSystemdMock`).
2. `OutputPath=systemd-services-meta-absent.tar.xz`.
3. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/systemd-services-meta-absent` (REQUIREMENT-DESIGN-systemd-services-meta.md).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.DryRunThenArchive = true
	req.OutputPath = "systemd-services-meta-absent.tar.xz"
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```