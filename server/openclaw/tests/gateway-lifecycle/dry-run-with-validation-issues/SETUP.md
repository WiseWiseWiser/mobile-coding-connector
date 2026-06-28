# Scenario

**Feature**: dry-run surfaces validation issues without starting

```
# invalid slack config adds Issues, does not start gateway
DryRun (missing bot token) -> issues contains validation error
```

## Steps

1. Seed slack enabled without bot token.
2. Call `OpDryRun`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDryRun
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.AppToken = "xapp-only"
	return nil
}
```