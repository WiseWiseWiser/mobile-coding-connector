# Scenario

**Feature**: missing scratch.json shows empty scratch pad on load

```
# scratch.json absent after harness reset
Run -> delete scratch.json -> FileTransferView

# page loads scratch section with empty textarea
Playwright -> GET /api/file-transfer/scratch -> empty content in UI
```

## Preconditions

1. `{AI_CRITIC_HOME}/file-transfer/scratch.json` is deleted before the browser opens.
2. Scratch pad UI is mounted above the file upload area.

## Steps

1. Set `Request.FileTransferScratchReset` to `true`.
2. Navigate to `/home/file-transfer`.
3. Wait for `[data-testid="file-transfer-scratch"]` and the scratch textarea.
4. Verify the textarea value is empty.

## Context

Missing `scratch.json` is treated as empty scratch (`content: ""`). The leaf
`Assert` also calls `GET /api/file-transfer/scratch` to verify the API contract.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferScratchReset = true
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```