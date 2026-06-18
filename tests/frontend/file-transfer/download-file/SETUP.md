# Scenario

**Feature**: download a pre-seeded file from the inbox

```
# storage contains hello.txt before UI opens
Run -> seed file-transfer/hello.txt

# user clicks Download on the row
Playwright -> GET /api/file-transfer/download?name=hello.txt -> browser save
```

## Preconditions

1. `{AI_CRITIC_HOME}/file-transfer/hello.txt` is seeded before the script runs.
2. The File Transfer list shows a row for `hello.txt` with a Download action.

## Steps

1. Seed `hello.txt` from `testdata/hello.txt`.
2. Open `/home/file-transfer` and wait for the `hello.txt` row.
3. Click Download and capture the browser download suggested filename.

## Context

Download is keyed by stored filename (`hello.txt`). The browser should save
the file under that name via `Content-Disposition: attachment`.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.FileTransferSeeds = []FileTransferSeed{
		{Name: "hello.txt", SourcePath: "testdata/hello.txt"},
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```