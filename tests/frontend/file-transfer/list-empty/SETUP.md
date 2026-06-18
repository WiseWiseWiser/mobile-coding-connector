# Scenario

**Feature**: empty file-transfer directory shows empty state

```
# storage has no uploaded files
Run -> empty file-transfer/ -> FileTransferView lists zero files

# user opens inbox
Playwright -> /home/file-transfer -> empty-state message
```

## Preconditions

1. `{AI_CRITIC_HOME}/file-transfer/` is reset to an empty directory before the script runs.
2. No files exist in the transfer store.

## Steps

1. Set `Request.FileTransferReset` to `true`.
2. Navigate to `/home/file-transfer`.
3. Verify the empty-state message is visible and the file row count is zero.

## Context

The empty state copy is: "No files yet — upload a file to get started". The
test accepts case-insensitive partial matching.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferReset = true
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```