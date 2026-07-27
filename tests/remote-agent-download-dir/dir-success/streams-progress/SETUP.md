# Scenario

**Feature**: directory download streams incremental stdout with per-item index and overall rollup

```
# multi-file remoteDir -> per-file GET with onProgress -> [N/M] + overall lines before summary
remote a.txt + sub/b.txt -> remote-agent download -> stdout streams progress then Download complete
```

## Preconditions

Remote tree at `uploads/stream-mirror`; local destination absent.

## Steps

1. Seed standard remote tree at `uploads/stream-mirror`.
2. Args: `download uploads/stream-mirror ./local-stream`.

## Context

REQUIREMENT leaf #3 — dir-success/streams-progress.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// L3 smoke: directory download progress via product binaries.
	req.UseCLI = true
	req.ServerPreseedFiles = map[string]string{
		"uploads/stream-mirror/a.txt":     "alpha\n",
		"uploads/stream-mirror/sub/b.txt": "bravo\n",
	}
	setDownloadArgs(t, req, "uploads/stream-mirror", "./local-stream")
	req.LocalDir = localDirRel("uploads/stream-mirror", "./local-stream")
	return nil
}
```