# Scenario

**Feature**: upload a file via UI and see it in the list

```
# storage starts empty
Run -> empty file-transfer/

# user uploads sample.txt through FileTransferView
Playwright -> choose file -> POST /api/file-transfer/upload -> row at top of list
```

## Preconditions

1. `{AI_CRITIC_HOME}/file-transfer/` is reset to empty before the script runs.
2. Fixture file `testdata/sample.txt` exists beside this leaf.

## Steps

1. Set `Request.FileTransferReset` to `true` and `Request.TimeoutSecs` to `120`.
2. Open `/home/file-transfer`.
3. Upload `testdata/sample.txt` via the file input (button or drag-and-drop target).
4. Wait for a list row containing `sample.txt` and a human-readable size.

## Context

The upload uses the hidden `<input type="file">` exposed by the upload area.
`CASE_DIR` injected by root `Run` resolves the fixture path for Playwright
`setInputFiles`.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferReset = true
	req.TimeoutSecs = 120
	return nil
}
```