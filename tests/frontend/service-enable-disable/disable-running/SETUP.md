# Scenario

**Feature**: Disable button on a running service shows won't-stop modal

```
# API seeds running enabled service before browser opens
POST /api/services -> POST /api/services/start -> running

# UI Disable opens ConfirmModal with deferred-stop message
Playwright -> Disable -> modal message; service still running
```

## Preconditions

1. One service is running before the Playwright script starts.
2. Service card shows a **Disable** button (enabled !== false).

## Steps

1. Set `ServiceSeed.Prepare` to `running-enabled`.
2. Script opens `/home/service`, clicks Disable, confirms, and captures modal text.

## Context

Sibling `enable-stopped/` covers the Enable path.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ServiceSeed = &ServiceSeed{
		ID:                "ui-dis-run-001",
		Name:              "ui-disable-running",
		Command:           "sleep 300",
		Prepare:           "running-enabled",
		StartBeforeScript: true,
	}
	return nil
}
```