# Scenario

**Feature**: defaults when openclaw.json is missing

```
# absent file yields default gateway port, slack disabled
Config store (missing) -> LoadConfig -> defaults
```

## Steps

1. Do not write initial config.
2. Load defaults via `OpLoadDefaults`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpLoadDefaults
	return nil
}
```