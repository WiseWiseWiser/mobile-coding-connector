# Scenario

**Feature**: seeded scratch.json populates textarea on page load

```
# harness writes scratch.json before browser opens
Run -> scratch.json (known content) -> FileTransferView

# mount GET loads scratch into textarea
Playwright -> /home/file-transfer -> textarea shows seeded text
```

## Preconditions

1. `{AI_CRITIC_HOME}/file-transfer/scratch.json` is seeded with known UTF-8 content.
2. Explicit Save is not required for display — load-only via GET on mount.

## Steps

1. Set `Request.FileTransferScratchSeed` to content `seeded-scratch-content-for-display`.
2. Navigate to `/home/file-transfer`.
3. Wait for the scratch textarea and read its value.

## Context

The harness writes `scratch.json` with `content` and `updated_at` (RFC3339 UTC).
The Playwright script compares the textarea value to the same seed string.

```go
import (
	"testing"
)

const scratchDisplaySeed = "seeded-scratch-content-for-display"

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferScratchSeed = &ScratchSeed{Content: scratchDisplaySeed}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```