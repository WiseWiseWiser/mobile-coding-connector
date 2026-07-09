# Scenario

**Feature**: single-file download unchanged shape

```
# seed uploads/hello.txt on server -> download ./hello.txt -> local bytes match
remote hello.txt -> remote-agent download -> hello.txt in agentWorkDir
```

## Preconditions

Remote file `uploads/hello.txt` exists on server; local destination absent.

## Steps

1. Copy `testdata/hello.txt` content into `serverHome` at `uploads/hello.txt`.
2. Args: `download uploads/hello.txt ./hello.txt`.

## Context

REQUIREMENT leaf #1 — file-regression/single-file. Assert only behaviors that
already work today; new retry/resume/streaming refinements are not asserted here.

```go
import (
	"os"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	content, err := os.ReadFile("testdata/hello.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	req.ServerPreseedFiles = map[string]string{
		"uploads/hello.txt": string(content),
	}
	setDownloadArgs(t, req, "uploads/hello.txt", "./hello.txt")
	return nil
}
```