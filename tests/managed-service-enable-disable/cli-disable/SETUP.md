# Scenario

**Feature**: `remote-agent service disable` prints contextual message

```
# running enabled service via API pre-start
POST /api/services/start -> running

# CLI disable prints API message to stdout
remote-agent service disable <name> -> exit 0 + prompt
```

## Preconditions

1. `remote-agent` is built and configured with test server URL/token.
2. Target service is running before CLI disable.

## Steps

1. Seed enabled `sleep` service.
2. Pre-start via API.
3. Run `remote-agent service disable <name>` (`Request.UseCLI=true`).

## Context

Sibling `cli-enable/` covers the enable subcommand.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("svc-cli-dis-001", "cli-disable-target", boolPtr(true)),
	}
	req.TargetID = "svc-cli-dis-001"
	req.Action = "disable"
	req.UseCLI = true
	req.PreStartID = "svc-cli-dis-001"
	req.CLIArgs = []string{"service", "disable", "cli-disable-target"}
	return nil
}
```