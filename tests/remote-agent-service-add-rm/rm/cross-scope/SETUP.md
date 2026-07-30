# Scenario

**Feature**: service rm resolves name across projectDir via ListAll

```
seed cross-scope-svc under non-default projectDir
  -> service rm cross-scope-svc
  -> resolves via all=1; Removed; gone
```

## Preconditions

1. Service lives under a projectDir that is **not** the server default scope.
2. Today `resolveServiceTarget` uses scoped `ListServices("")` → would miss;
   implementer must switch to list-all (proves this leaf).

## Steps

1. Seed service with `ProjectDir = t.TempDir()` (other scope).
2. CLI: `service rm cross-scope-svc` (no --project-dir).
3. Assert Removed and gone from ListAll.

## Context

REQUIREMENT leaf: `rm/cross-scope`. Primary proof that name resolution uses ListAll.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	other := t.TempDir()
	req.OtherProjectDir = other
	req.Services = []ServiceSeed{
		sleepService("svc-cross-001", "cross-scope-svc", other),
	}
	req.TargetID = "svc-cross-001"
	req.TargetName = "cross-scope-svc"
	setCLI(req, "service", "rm", "cross-scope-svc")
	return nil
}
```
