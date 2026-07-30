# Scenario

**Feature**: service list --all shows services across projectDirs

```
seed local-web @ LOCAL + other-api @ OTHER
  -> service list --all
  -> stdout includes both names
```

## Steps

1. Seed two services with different absolute projectDirs.
2. CLI: `service list --all`.

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
	setCLI(req, "service", "list", "--all")
	return nil
}
```
