# Scenario

**Feature**: slack disabled allows start without tokens

```
# validation passes when slack is off
ValidateStartConfig (slack off) -> nil -> POST start 200
```

## Steps

1. Use default config (slack disabled).
2. POST /api/openclaw/start.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIStart
	return nil
}
```