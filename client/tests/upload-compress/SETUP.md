# Scenario

**Feature**: whole-file gzip upload with deterministic resume

```
local file -> gzip (optional) -> Client.UploadFile -> init(compressed) -> chunks -> complete(gunzip)
```

## Preconditions

- `client.UploadFile` honors `UploadOptions.NoCompress`.
- Deterministic gzip yields a stable `file_hash` for resume.

## Steps

1. Leaf `Setup` sets content / NoCompress / PrefilledChunks.
2. Root `Run` uploads via mock server that gunzips when compressed.

## Context

In-process transport tests — no real remote-agent binary.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.MaxChunkAttempts == 0 {
		req.MaxChunkAttempts = 3
	}
	return nil
}
```
