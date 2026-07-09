# Scenario

**Feature**: download resumes partial local file via HTTP Range

```
# local file pre-filled to offset N; client sends Range: bytes=N-
Client.DownloadFile -> Range GET -> append -> full bytes
```

## Preconditions

8 KiB remote file; local file pre-filled with first 4 KiB.

## Steps

1. Set `FileSize=8192`, `LocalPrefillBytes=4096`.

## Context

Models user re-running download after interrupt at 50%.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FileSize = 8192
	req.LocalPrefillBytes = 4096
	req.AlwaysFail = false
	req.TransientFails = 0
	return nil
}
```