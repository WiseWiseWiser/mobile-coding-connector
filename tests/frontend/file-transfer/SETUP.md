# Scenario

**Feature**: File Transfer inbox operations (list, upload, download, delete)

```
# quick-test serves FileTransferView backed by /api/file-transfer
Run -> {AI_CRITIC_HOME}/file-transfer/ (reset/seed) -> FileTransferView

# Playwright exercises inbox actions; Assert checks UI and API side effects
leaf script.js -> upload/download/remove -> ScriptResult -> Assert
```

## Preconditions

1. Quick-test server is running with an isolated temp `AI_CRITIC_HOME`.
2. Dedicated `/api/file-transfer` endpoints are registered on the server.
3. `FileTransferView` is reachable at `/home/file-transfer`.

## Steps

1. Each child leaf configures `Request.FileTransferReset` and/or `Request.FileTransferSeeds`.
2. Root `Run` prepares `{AI_CRITIC_HOME}/file-transfer/` after the server is healthy.
3. The leaf `script.js` opens the File Transfer page and performs the operation under test.
4. The leaf `Assert` verifies `ScriptResult` and, for delete, the API list.

## Context

File Transfer storage is global under `{AI_CRITIC_HOME}/file-transfer/` (flat, no
nested folders in v1). Leaves isolate storage by resetting the directory or
seeding known files before the browser script runs.

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