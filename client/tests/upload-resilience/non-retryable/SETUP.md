# Scenario

**Feature**: non-retryable 4xx fails immediately

```
# chunk 1 returns 400 once; client does not retry
Client.UploadFile -> chunk[0] ok -> chunk[1] x1 -> error
```

## Preconditions

Five-chunk file; chunk 1 returns HTTP 400.

## Steps

1. Set `FlakyChunkIndex=1`, `PermanentStatus=400`, `TransientFails=0`, `AlwaysFailChunk=-1`.

## Context

4xx (except 429) must not trigger backoff retries.

```go
import (
	"net/http"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.TotalBytes = 5 * 1024 * 1024
	req.FlakyChunkIndex = 1
	req.PermanentStatus = http.StatusBadRequest
	req.TransientFails = 0
	req.AlwaysFailChunk = -1
	req.MaxChunkAttempts = 5
	return nil
}
```