# Scenario

**Feature**: stop is idempotent when already stopped

```
# Stop on stopped gateway succeeds without error
Stop -> Stop -> still stopped, no error
```

## Steps

1. Call `OpStop` twice (SecondStart flag = double stop).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpStop
	req.SecondStart = true
	return nil
}
```