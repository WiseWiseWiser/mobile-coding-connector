# Scenario

**Feature**: download survives transient 502

```
# first two GETs return 502, third succeeds; full file written
Client.DownloadFile -> GET x3 -> bytes match
```

## Preconditions

4 KiB file. First two GET attempts fail with 502.

## Steps

1. Set `TransientFails=2`, `FailStatus=502`, `MaxDownloadAttempts=5`.

## Context

Proves retry re-issues GET without discarding partial progress.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FileSize = 4096
	req.TransientFails = 2
	req.FailStatus = 502
	req.MaxDownloadAttempts = 5
	req.AlwaysFail = false
	return nil
}
```