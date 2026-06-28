# Scenario

**Feature**: slack disabled skips slack-specific checks

```
# slack_enabled check is skip when slack off
Doctor (slack off) -> slack_enabled (skip)
```

## Steps

1. Default config (slack disabled).
2. Run doctor.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDoctor
	return nil
}
```