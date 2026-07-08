# Scenario

**Bug**: re-uploading same binary re-sends every chunk

```
# server already has chunks 0..38; client should upload only chunk 39
cached chunks -> UploadFile -> single POST for missing chunk
```

## Preconditions

40-chunk file; server cache prefilled with first 39 chunks (73% progress).

## Steps

1. Set `PrefilledChunks=39`, `TotalBytes=40 * ChunkSize`.

## Context

Models user re-running `remote-agent upload` after failure at ~73%.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func Setup(t *testing.T, req *Request) error {
	req.TotalBytes = 40 * int64(client.ChunkSize)
	req.PrefilledChunks = 39
	req.AlwaysFailChunk = -1
	return nil
}
```