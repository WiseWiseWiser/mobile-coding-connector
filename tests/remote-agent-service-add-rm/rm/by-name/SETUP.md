# Scenario

**Feature**: service rm by name

```
seed svc-rm-name-001 (rm-by-name-target)
  -> service rm rm-by-name-target
  -> Removed; gone from ListAll / disk
```

## Steps

1. Seed one service.
2. CLI: `service rm rm-by-name-target`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Empty ProjectDir → normalized to server default scope so by-name does not
	// depend on ListAll (cross-scope is covered by rm/cross-scope).
	req.Services = []ServiceSeed{
		sleepService("svc-rm-name-001", "rm-by-name-target", ""),
	}
	req.TargetID = "svc-rm-name-001"
	req.TargetName = "rm-by-name-target"
	setCLI(req, "service", "rm", "rm-by-name-target")
	return nil
}
```
