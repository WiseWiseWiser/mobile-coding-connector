# Scenario

**Feature**: interrupted gzip upload resumes from cached wire chunks

```
prefill all gzip wire chunks -> 0 chunk POSTs -> complete original bytes
```

Highly compressible payloads often fit in one 2 MiB wire chunk after gzip; resume
still skips that cached chunk via the deterministic `file_hash`. Multi-chunk skip
behavior for the same protocol is covered by `upload-resilience/cross-run-resume`
(raw path).

```go
import (
	"bytes"
	"testing"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Content = bytes.Repeat([]byte("compress-me-please-"), 64*1024)
	req.NoCompress = false
	wire, compressed, err := prepareWire(req.Content, false)
	if err != nil {
		t.Fatal(err)
	}
	if !compressed {
		t.Fatal("expected compressible content")
	}
	totalChunks := (len(wire) + client.ChunkSize - 1) / client.ChunkSize
	if totalChunks < 1 {
		totalChunks = 1
	}
	req.PrefilledChunks = totalChunks
	return nil
}
```
