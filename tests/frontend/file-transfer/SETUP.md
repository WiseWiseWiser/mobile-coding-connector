# Scenario

**Feature**: File Transfer scratch pad and inbox operations

```
# quick-test serves FileTransferView backed by /api/file-transfer + scratch API
Run -> file-transfer/ (reset/seed files + scratch.json) -> FileTransferView

# Playwright exercises scratch save/copy and inbox actions
leaf script.js -> scratch/inbox actions -> ScriptResult -> Assert
```

## Preconditions

1. Quick-test server is running with an isolated temp `AI_CRITIC_HOME`.
2. Dedicated `/api/file-transfer` endpoints are registered on the server.
3. `FileTransferView` is reachable at `/home/file-transfer`.

## Steps

1. Each child leaf configures `Request.FileTransferReset`, `Request.FileTransferSeeds`,
   `Request.FileTransferScratchReset`, and/or `Request.FileTransferScratchSeed`.
2. Root `Run` prepares `{AI_CRITIC_HOME}/file-transfer/` and `scratch.json` after the server is healthy.
3. The leaf `script.js` opens the File Transfer page and performs the operation under test.
4. The leaf `Assert` verifies `ScriptResult` and, where needed, scratch or file-list API state.

## Context

File Transfer storage is global under `{AI_CRITIC_HOME}/file-transfer/` (flat, no
nested folders in v1). Scratch content lives in `scratch.json` beside uploaded
files. Leaves isolate storage by resetting the directory or seeding known files
and scratch content before the browser script runs. Scratch UI selectors:
`file-transfer-scratch`, `file-transfer-scratch-input`, `file-transfer-scratch-save`,
`file-transfer-scratch-copy`.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if req.ScriptPath == "" {
		req.ScriptPath = "script.js"
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```