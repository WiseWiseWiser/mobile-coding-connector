# Scenario

**Feature**: stopped gateway warns with start hint

```
# gateway not running -> gateway_running warn + hint
Doctor (stopped) -> gateway_running (warn)
```

## Steps

1. Default config, gateway not started.
2. Run doctor.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDoctor
	return nil
}
```