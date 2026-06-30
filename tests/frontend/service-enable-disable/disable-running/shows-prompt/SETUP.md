# Scenario

**Feature**: Disable modal shows won't-stop-immediately message for running service

```
# running service card on /home/service
Playwright -> click Disable -> ConfirmModal message

# process must remain running after confirm
GET /api/services -> pid > 0
```

## Preconditions

Parent `disable-running` seeds a running enabled service.

## Steps

1. Inherit parent `ServiceSeed`.
2. Script asserts modal message and post-action running state.

## Context

REQUIREMENT leaf: `disable-running/shows-prompt`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TimeoutSecs = 120
	if req.ServiceSeed == nil {
		req.ServiceSeed = &ServiceSeed{
			ID:      "ui-dis-run-001",
			Name:    "ui-disable-running",
			Command: "sleep 300",
			Prepare: "running-enabled",
		}
	}
	return nil
}
```