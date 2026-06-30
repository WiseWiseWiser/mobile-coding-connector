# Scenario

**Feature**: Enable button on disabled stopped service shows daemon-check modal

```
# API seeds disabled stopped service
POST /api/services -> POST stop -> POST disable

# UI Enable opens ConfirmModal with daemon message
Playwright -> Enable -> modal message; enabled badge updates
```

## Preconditions

1. Service is disabled (`enabled=false`) and stopped before the script runs.
2. Service card shows an **Enable** button.

## Steps

1. Set `ServiceSeed.Prepare` to `stopped-disabled`.
2. Script opens `/home/service`, clicks Enable, confirms, and checks UI/API state.

## Context

Sibling `disable-running/` covers the Disable path.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ServiceSeed = &ServiceSeed{
		ID:      "ui-en-stop-001",
		Name:    "ui-enable-stopped",
		Command: "sleep 300",
		Prepare: "stopped-disabled",
	}
	return nil
}
```