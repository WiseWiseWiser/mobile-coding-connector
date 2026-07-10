# Scenario

**Feature**: Backup Now still allowed when periodic task is enabled

```
CanRunBackupNow(hasEndpoint=true, running=false, server="foo.example.com") -> true
# enabled=true does not change the pure helper (not an input)
```

## Preconditions

Same ready conditions as when-disabled; task may be on for hourly schedule.

## Steps

1. Ready inputs; document Enabled=true.

## Context

REQUIREMENT #2.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.HasEndpoint = true
	req.Running = false
	req.ServerName = "foo.example.com"
	req.Enabled = true // story only; not passed to helper
	return nil
}
```
