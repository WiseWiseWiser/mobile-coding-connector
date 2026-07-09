# Scenario

**Feature**: download retries transient GET failures and resumes via Range

```
# client downloads file via GET; mock server may inject failures or honor Range
local file (optional prefill) -> Client.DownloadFile -> GET(s) with retry -> bytes on disk
```

## Preconditions

- `client.DownloadFile` and `client.DownloadOptions` are exported.
- `DownloadOptions.Retry` configures max attempts and zero-delay backoff for tests.
- Mock server tracks GET attempt count and `Range` headers.

## Steps

1. Leaf `Setup` sets failure injection or prefill fields on `Request`.
2. Root `Run` starts mock server, calls `DownloadFile`.

## Context

Transport-layer unit tests — no real `remote-agent` or `server/filedownload` process.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.FileSize <= 0 {
		req.FileSize = 4096
	}
	if req.MaxDownloadAttempts == 0 {
		req.MaxDownloadAttempts = 5
	}
	return nil
}
```