# Scenario

**Feature**: service list --project-dir scopes to one project

```
seed web @ LOCAL + api @ OTHER
  -> service list --project-dir LOCAL
  -> stdout shows web; not api
```

## Steps

1. Seed two projectDirs.
2. CLI: `service list --project-dir <LOCAL>` (no `--all`).

## Context

Contrasts with `list/all`. Uses explicit `--project-dir` for deterministic L2
scope (avoids process cwd as default normalizeProjectDir target).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	local := t.TempDir()
	other := t.TempDir()
	req.LocalProjectDir = local
	req.OtherProjectDir = other
	req.Services = []ServiceSeed{
		sleepService("local-web", "web", local),
		sleepService("other-api", "api", other),
	}
	setCLI(req, "service", "list", "--project-dir", local)
	return nil
}
```
