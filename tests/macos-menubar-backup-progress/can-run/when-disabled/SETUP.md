# Scenario

**Feature**: Backup Now allowed when periodic task is disabled

```
# task enabled=false does not block one-shot
CanRunBackupNow(hasEndpoint=true, running=false, server="foo.example.com") -> true
```

## Preconditions

Endpoint configured, not already running, non-empty server name. Task is off
(Enable only gates hourly schedule, not Backup Now).

## Steps

1. Ready inputs; document Enabled=false.

## Context

REQUIREMENT #1, #14 — one-shot independent of Enable.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.HasEndpoint = true
	req.Running = false
	req.ServerName = "foo.example.com"
	req.Enabled = false // story only; not passed to helper
	return nil
}
```
