# Scenario

**Feature**: boot auto-start skips disabled service definitions

```
# mixed services.json before server boot
enabled svc (field absent) + disabled svc (enabled=false)

# AutoStartConfiguredServices on boot
server boot -> only enabled service gets a live PID
```

## Preconditions

1. `services.json` is written before the server subprocess starts.
2. One definition omits `enabled` (defaults true); another sets `enabled: false`.
3. Both commands are long-running `sleep` processes.

## Steps

1. Seed `enabled-svc` (no `enabled` field) and `disabled-svc` (`enabled: false`).
2. Start server (triggers `AutoStartConfiguredServices`).
3. Wait 3 seconds for boot auto-start to settle.
4. Inspect `GET /api/services` for each id.

## Context

REQUIREMENT leaf: `autostart-skips-disabled/`. Proves `enabled` defaults to true
when the field is absent.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Services = []ServiceSeed{
		{ID: "enabled-svc", Name: "enabled-svc", Command: "sleep 300"},
		sleepService("disabled-svc", "disabled-svc", boolPtr(false)),
	}
	req.Action = "boot-only"
	req.WaitAfterSecs = 3
	return nil
}
```