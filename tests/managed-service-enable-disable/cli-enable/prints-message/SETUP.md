# Scenario

**Feature**: CLI enable prints daemon-check prompt for stopped service

```
# remote-agent service enable on stopped disabled service
remote-agent -> POST /api/services/enable -> stdout daemon message
```

## Preconditions

Parent `cli-enable` leaves the service stopped at CLI invocation time.

## Steps

1. Inherit parent CLI enable setup.
2. Assert exit 0 and stdout daemon prompt.

## Context

REQUIREMENT leaf: `cli-enable/prints-message`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = true
	req.WaitAfterSecs = 0
	if len(req.CLIArgs) == 0 {
		req.CLIArgs = []string{"service", "enable", "cli-enable-target"}
	}
	return nil
}
```