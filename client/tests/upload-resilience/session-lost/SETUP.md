# Scenario

**Bug**: server drops in-memory upload session mid-transfer (404)

```
# chunks 0..28 stored; session invalidated; chunk 29 must recover via re-init
Client.UploadFile -> chunk[29] 404 -> re-init -> finish upload
```

## Preconditions

40-chunk file mirroring spl-linux-amd64 shape; session dies after chunk index 28.

## Steps

1. Set `SessionDropAfterChunk=28`, `TotalBytes=40 * ChunkSize`.

## Context

Reproduces keep-alive restart / server reload during long unstable-network upload.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func Setup(t *testing.T, req *Request) error {
	req.TotalBytes = 40 * int64(client.ChunkSize)
	req.SessionDropAfterChunk = 28
	req.AlwaysFailChunk = -1
	req.MaxChunkAttempts = 5
	return nil
}
```