# Scenario

**Feature**: chunked upload retries transient per-chunk failures

```
# client uploads file in chunks; mock server may inject failures per index
local file -> Client.UploadFile -> init -> chunk POSTs (with retry) -> complete -> assembled bytes
```

## Preconditions

- `client.UploadFile` and `client.ChunkSize` are exported.
- `UploadOptions.ChunkRetry` configures max attempts and zero-delay backoff for tests.
- Mock server tracks per-chunk POST counts and assembles bytes on complete.

## Steps

1. Leaf `Setup` sets failure injection fields on `Request`.
2. Root `Run` writes temp file, starts mock server, calls `UploadFile`.

## Context

Transport-layer unit tests — no real `remote-agent` or `server/fileupload` process.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.TotalBytes <= 0 {
		req.TotalBytes = 5 * 1024 * 1024
	}
	if req.MaxChunkAttempts == 0 {
		req.MaxChunkAttempts = 5
	}
	if req.AlwaysFailChunk == 0 && req.TransientFails == 0 && req.TransportFailCount == 0 {
		// Go zero default; -1 means "no always-fail chunk" unless a leaf sets >=0.
		req.AlwaysFailChunk = -1
	}
	return nil
}
```