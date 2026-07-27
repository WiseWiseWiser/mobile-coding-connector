# Scenario

**Feature**: Backup Now blocked while a run is in progress

```
CanRunBackupNow(hasEndpoint=true, running=true, server="foo.example.com") -> false
```

## Preconditions

A backup is already running (no overlapping jobs).

## Steps

1. Running=true; endpoint and server present.

## Context

REQUIREMENT #4.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.HasEndpoint = true
	req.Running = true
	req.ServerName = "foo.example.com"
	return nil
}
```
