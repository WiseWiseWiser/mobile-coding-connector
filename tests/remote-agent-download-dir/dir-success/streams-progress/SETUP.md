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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ServerPreseedFiles = map[string]string{
		"uploads/stream-mirror/a.txt":     "alpha\n",
		"uploads/stream-mirror/sub/b.txt": "bravo\n",
	}
	setDownloadArgs(t, req, "uploads/stream-mirror", "./local-stream")
	req.LocalDir = localDirRel("uploads/stream-mirror", "./local-stream")
	return nil
}
```