# Scenario

**Feature**: backup --show-config merges CLI --exclude into effective preview

```
# effective = Merge(builtin, remote backup-config.json, CLI flags)
--show-config --exclude .knowledge-index -> JSON lists path with reason user excluded
```

## Preconditions

Default `serverHome` fixtures; server serves GET backup-config with optional query params.

## Steps

1. `ShowConfig=true`, `ExcludePaths=[".knowledge-index"]`.
2. Args: `machine backup`.

## Context

REQUIREMENT leaf `backup/show-config-cli-exclude`. CLI `--exclude` on preview only; no archive.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ShowConfig = true
	req.ExcludePaths = []string{".knowledge-index"}
	req.Args = []string{"machine", "backup"}
	return nil
}
```