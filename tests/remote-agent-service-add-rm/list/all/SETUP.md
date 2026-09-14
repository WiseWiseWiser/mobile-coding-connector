# Scenario

**Feature**: service list --all shows all services

```
seed local-web + other-api
  -> service list --all
  -> stdout includes both names
```

## Steps

1. Seed two services.
2. CLI: `service list --all`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("local-web", "web"),
		sleepService("other-api", "api"),
	}
	setCLI(req, "service", "list", "--all")
	return nil
}
```
