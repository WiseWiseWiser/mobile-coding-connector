# Scenario

**Feature**: `remote-agent service enable` prints contextual message

```
# stopped disabled service
services.json(enabled=false) -> server boot -> stopped

# CLI enable prints API message to stdout
remote-agent service enable <name> -> exit 0 + daemon prompt
```

## Preconditions

1. Target service is disabled and stopped.
2. `remote-agent` calls the enable API endpoint.

## Steps

1. Seed disabled `sleep` service (not pre-started).
2. Run `remote-agent service enable <name>` (`Request.UseCLI=true`).

## Context

Sibling `cli-disable/` covers disable subcommand.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Services = []ServiceSeed{
		sleepService("svc-cli-en-001", "cli-enable-target", boolPtr(false)),
	}
	req.TargetID = "svc-cli-en-001"
	req.Action = "enable"
	req.UseCLI = true
	req.CLIArgs = []string{"service", "enable", "cli-enable-target"}
	req.WaitAfterSecs = 0
	return nil
}
```