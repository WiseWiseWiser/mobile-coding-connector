# Scenario

**Feature**: download aborts when retries exhausted

```
# every GET returns 502; client retries up to MaxDownloadAttempts then fails
Client.DownloadFile -> GET xN -> error
```

## Preconditions

4 KiB file; every GET permanently fails.

## Steps

1. Set `AlwaysFail=true`, `FailStatus=502`, `MaxDownloadAttempts=3`.

## Context

Ensures bounded retry does not loop forever.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FileSize = 4096
	req.AlwaysFail = true
	req.FailStatus = 502
	req.MaxDownloadAttempts = 3
	return nil
}
```