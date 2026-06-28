# Scenario

**Feature**: doctor reports node and openclaw CLI on PATH

```
# runtime dependency checks reflect host environment
Doctor -> node (ok|fail+hint), openclaw_cli (ok|fail+hint)
```

## Steps

1. Run doctor with default config.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDoctor
	return nil
}
```