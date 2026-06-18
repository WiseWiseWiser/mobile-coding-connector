# Scenario

**Feature**: remove a file from the inbox with confirmation

```
# storage contains temp.txt
Run -> seed file-transfer/temp.txt

# user confirms Remove on the row
Playwright -> confirm dialog -> DELETE /api/file-transfer -> row disappears
```

## Preconditions

1. `{AI_CRITIC_HOME}/file-transfer/temp.txt` is seeded before the script runs.
2. Remove shows a confirmation dialog before deleting.

## Steps

1. Seed `temp.txt` from `testdata/temp.txt`.
2. Open `/home/file-transfer` and wait for the `temp.txt` row.
3. Click Remove, accept the confirmation dialog, and wait for the row to disappear.

## Context

After UI delete, `GET /api/file-transfer` must no longer list `temp.txt`. The
leaf `Assert` checks both `ScriptResult` and the API list.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferSeeds = []FileTransferSeed{
		{Name: "temp.txt", SourcePath: "testdata/temp.txt"},
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```