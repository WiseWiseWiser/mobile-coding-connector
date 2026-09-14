# Scenario

**Feature**: plain service list (no --all) shows all services

```
seed web + api
  -> service list
  -> stdout shows both
```

## Steps

1. Seed two services.
2. CLI: `service list` (no `--all`).

## Context

Contrasts with older scoped behavior; list is always global now.

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
	setCLI(req, "service", "list")
	return nil
}
```
