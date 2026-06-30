# Scenario

**Feature**: Enable modal shows daemon-check message for disabled stopped service

```
# disabled stopped service card
Playwright -> click Enable -> ConfirmModal daemon message

# UI reflects enabled state after confirm
badge or Enable->Disable button swap; apiEnabled true
```

## Preconditions

Parent `enable-stopped` seeds a disabled stopped service via API.

## Steps

1. Inherit parent `ServiceSeed`.
2. Script asserts modal message and post-confirm enabled UI markers.

## Context

REQUIREMENT leaf: `enable-stopped/shows-prompt`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TimeoutSecs = 120
	if req.ServiceSeed == nil {
		req.ServiceSeed = &ServiceSeed{
			ID:      "ui-en-stop-001",
			Name:    "ui-enable-stopped",
			Command: "sleep 300",
			Prepare: "stopped-disabled",
		}
	}
	return nil
}
```