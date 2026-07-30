# Scenario

**Feature**: service add --disabled persists enabled=false

```
service add --name demo-disabled --command "sleep 300" --disabled
  -> enabled false on disk; not started
```

## Steps

1. Run add with `--disabled`.
2. Assert disk `enabled: false` and not running.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.TargetName = "demo-disabled"
	setCLI(req,
		"service", "add",
		"--name", "demo-disabled",
		"--command", "sleep 300",
		"--disabled",
	)
	return nil
}
```
