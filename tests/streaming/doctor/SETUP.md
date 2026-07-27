# Scenario

**Feature**: remote-agent ws-proxy doctor streams server checks

```
# CLI uses /api/ws-proxy/doctor/stream (not buffered JSON /doctor)
remote-agent ws-proxy doctor -> streamcmd -> incremental stdout
```

## Preconditions

Server running with seeded ws-proxy config and stubbed external checks.

## Steps

1. Enable `RecordLineTimes` for timing-sensitive leaves.

## Context

Groups all doctor integration leaves.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.RecordLineTimes = true
	return nil
}
```
