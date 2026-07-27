# Scenario

**Feature**: dry-run summary lists INSTALLED SOFTWARE and ENV meta sections

```
# default serverHome (no git seed) -> backup --dry-run -> INSTALLED + ENV tables before TOTAL
```

## Preconditions

Default `serverHome` fixtures. Server subprocess inherits real `PATH` for installed-tool
snapshot (NAME rows may vary by environment).

## Steps

1. Args: `machine backup --dry-run`.

## Context

REQUIREMENT leaf `backup/dry-run-meta` (REQUIREMENT-DESIGN-dry-run-meta-tables.md).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"machine", "backup", "--dry-run"}
	return nil
}
```