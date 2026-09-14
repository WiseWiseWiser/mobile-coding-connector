# Scenario

**Feature**: service rm resolves name from global list

```
seed cross-scope-svc
  -> service rm cross-scope-svc
  -> Removed; gone
```

## Steps

1. Seed service.
2. CLI: `service rm cross-scope-svc`.
3. Assert Removed and gone from List.

## Context

REQUIREMENT leaf: `rm/cross-scope` (legacy name; services are global).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("svc-cross-001", "cross-scope-svc"),
	}
	req.TargetID = "svc-cross-001"
	req.TargetName = "cross-scope-svc"
	setCLI(req, "service", "rm", "cross-scope-svc")
	return nil
}
```
