# Scenario

**Feature**: service rm by id

```
seed svc-rm-id-001 -> service rm svc-rm-id-001 -> Removed; gone
```

## Steps

1. Seed one service with fixed id.
2. CLI: `service rm svc-rm-id-001`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Empty ProjectDir → default scope; id match does not need ListAll.
	req.Services = []ServiceSeed{
		sleepService("svc-rm-id-001", "rm-by-id-target", ""),
	}
	req.TargetID = "svc-rm-id-001"
	req.TargetName = "rm-by-id-target"
	setCLI(req, "service", "rm", "svc-rm-id-001")
	return nil
}
```
