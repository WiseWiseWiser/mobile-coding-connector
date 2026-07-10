# Scenario

**Feature**: Backup Now blocked with empty server name

```
CanRunBackupNow(hasEndpoint=true, running=false, server="") -> false
```

## Preconditions

No server selected / empty scope key.

## Steps

1. ServerName empty; endpoint present; not running.

## Context

REQUIREMENT #5.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.HasEndpoint = true
	req.Running = false
	req.ServerName = ""
	return nil
}
```
