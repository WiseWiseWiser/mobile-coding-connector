# Scenario

**Feature**: CLI disable prints won't-stop-immediately prompt for running service

```
# remote-agent service disable on running service
remote-agent -> POST /api/services/disable -> stdout message
```

## Preconditions

Parent `cli-disable` has started the service and invoked CLI disable.

## Steps

1. Inherit parent CLI disable setup.
2. Assert exit 0 and stdout prompt.

## Context

REQUIREMENT leaf: `cli-disable/prints-message`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = true
	req.WaitAfterSecs = 0
	if len(req.CLIArgs) == 0 {
		req.CLIArgs = []string{"service", "disable", "cli-disable-target"}
	}
	return nil
}
```