# Scenario

**Feature**: upload survives transient 502 on one chunk

```
# chunk 2 fails twice with 502, succeeds on third POST; same upload_id throughout
Client.UploadFile -> chunk[2] x3 -> complete -> bytes match
```

## Preconditions

Five-chunk file (10 MiB). Chunk index 2 fails twice then succeeds.

## Steps

1. Set `FlakyChunkIndex=2`, `TransientFails=2`, `FailStatus=502`.

## Context

Proves retry reuses same session without restarting from chunk 0.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TotalBytes = 10 * 1024 * 1024 // 5 x 2 MiB chunks
	req.FlakyChunkIndex = 2
	req.TransientFails = 2
	req.FailStatus = 502
	req.MaxChunkAttempts = 5
	req.AlwaysFailChunk = -1
	return nil
}
```