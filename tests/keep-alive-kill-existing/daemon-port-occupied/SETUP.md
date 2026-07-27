# Scenario

**Feature**: kill daemon-port occupier before start

```
port occupier (23312) -> keep-alive --kill-existing -> occupier dead -> status API
```

## Preconditions

`KillExisting=true`, daemon port occupier only.

## Steps

1. Do not occupy server port.

## Context

Simulates stale keep-alive management listener.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.KillExisting = true
	req.OccupyServerPort = false
	req.OccupyDaemonPort = true
	req.ExpectStart = true
	req.ExpectError = false
	return nil
}
```