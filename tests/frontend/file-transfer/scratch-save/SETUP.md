# Scenario

**Feature**: explicit Save persists scratch textarea content via PUT

```
# scratch starts empty (no scratch.json)
Run -> delete scratch.json -> FileTransferView

# user types text and clicks Save
Playwright -> fill textarea -> Save -> PUT /api/file-transfer/scratch
```

## Preconditions

1. `scratch.json` is absent before the browser opens (`FileTransferScratchReset`).
2. Save is explicit — no auto-save on typing in v1.

## Steps

1. Set `Request.FileTransferScratchReset` to `true`.
2. Navigate to `/home/file-transfer`.
3. Fill the scratch textarea with `saved-from-playwright-scratch-test`.
4. Click `[data-testid="file-transfer-scratch-save"]`.
5. Verify the textarea still shows the saved text.

## Context

After Save, the leaf `Assert` calls `GET /api/file-transfer/scratch` to confirm
the server persisted the textarea value.

```go
import (
	"testing"
)

const scratchSaveText = "saved-from-playwright-scratch-test"

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferScratchReset = true
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```