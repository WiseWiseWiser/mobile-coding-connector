# Scenario

**Feature**: Copy button copies scratch textarea content to clipboard

```
# scratch.json seeded before browser opens
Run -> scratch.json (known content) -> FileTransferView

# user clicks Copy (no server round-trip)
Playwright -> Copy -> navigator.clipboard.readText()
```

## Preconditions

1. `{AI_CRITIC_HOME}/file-transfer/scratch.json` is seeded with `seeded-scratch-for-copy-test`.
2. Playwright grants `clipboard-read` and `clipboard-write` before reading clipboard.

## Steps

1. Set `Request.FileTransferScratchSeed` to `seeded-scratch-for-copy-test`.
2. Navigate to `/home/file-transfer`.
3. Wait for the scratch textarea to show seeded content.
4. Click `[data-testid="file-transfer-scratch-copy"]`.
5. Read clipboard via `page.evaluate(() => navigator.clipboard.readText())`.

## Context

Copy is client-side only — it copies the current textarea value without PUT.
The test verifies clipboard text matches the seeded scratch content.

```go
import (
	"testing"
)

const scratchCopySeed = "seeded-scratch-for-copy-test"

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferScratchSeed = &ScratchSeed{Content: scratchCopySeed}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```