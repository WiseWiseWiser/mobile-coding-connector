# Scenario

**Feature**: upload aborts when retries exhausted

```
# chunk 0 always 502; client retries up to MaxChunkAttempts then fails
Client.UploadFile -> chunk[0] xN -> error (no complete)
```

## Preconditions

Small file; chunk 0 permanently fails.

## Steps

1. Set `AlwaysFailChunk=0`, `MaxChunkAttempts=3`.

## Context

Ensures bounded retry does not loop forever.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TotalBytes = 3 * 1024 * 1024
	req.AlwaysFailChunk = 0
	req.FailStatus = 502
	req.MaxChunkAttempts = 3
	return nil
}
```