# Client Upload Compress Doctests

Whole-file gzip before chunking, with deterministic hash resume.

# DSN (Domain Specific Notion)

**Participants**

- **Client.UploadFile** — gzip (when smaller) → hash/chunk gzip bytes → init(`compressed`) → chunks → complete.
- **Mock upload server** — stores wire chunks; on complete gunzips when `compressed=true`.
- **Resume cache** — `received_chunks` keyed by hash of the wire (gzip) payload.

**Behaviors**

- Default compress when gzip shrinks the file; final remote bytes are original content.
- `--no-compress` / `NoCompress` uploads raw bytes with `compressed=false`.
- Interrupt/resume continues from cached gzip chunks (same deterministic hash).

## Version

0.0.1

## Decision Tree

```
[UploadFile compress]
 |
 +-- wire-format/
 |    |
 |    +-- gzip-when-smaller/     (LEAF)  compressed=true; final size = original
 |    +-- no-compress-flag/      (LEAF)  compressed=false; wire == raw
 |
 +-- resume/
      |
      +-- continues-from-cached-gzip-chunks/  (LEAF)  prefilled gzip chunks skipped
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `wire-format/gzip-when-smaller` | Compressible file uploads gzip; complete returns original size |
| 2 | `wire-format/no-compress-flag` | NoCompress forces raw wire bytes |
| 3 | `resume/continues-from-cached-gzip-chunks` | Most gzip chunks cached; only missing chunk POSTed |

## How to Run

```sh
doctest vet ./client/tests/upload-compress
doctest test ./client/tests/upload-compress/...
```

```go
import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/doctest/session"
)

type Request struct {
	// Content is the exact local file body. When empty, a compressible default is used.
	Content          []byte
	NoCompress       bool
	PrefilledChunks  int // wire chunks 0..N-1 already cached (gzip or raw per mode)
	MaxChunkAttempts int
}

type Response struct {
	UploadErr       string
	ResultSize      int64
	InitCompressed  *bool
	WireTotalSize   int64
	TotalChunkPosts int
	ChunkAttempts   map[int]int
	CompleteCalled  bool
	AssembledRaw    []byte
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	_ = d
	resp := &Response{ChunkAttempts: make(map[int]int)}
	if req.MaxChunkAttempts == 0 {
		req.MaxChunkAttempts = 3
	}
	content := req.Content
	if len(content) == 0 {
		// ~1.1 MiB compressible → typically 1–2 wire chunks after gzip.
		content = bytes.Repeat([]byte("compress-me-please-"), 64*1024)
	}

	dir := t.TempDir()
	localFile := filepath.Join(dir, "src.bin")
	if err := os.WriteFile(localFile, content, 0644); err != nil {
		t.Fatalf("write local: %v", err)
	}

	wire, compressedExpected, err := prepareWire(content, req.NoCompress)
	if err != nil {
		t.Fatalf("prepare wire: %v", err)
	}
	totalChunks := (len(wire) + client.ChunkSize - 1) / client.ChunkSize
	if totalChunks == 0 {
		totalChunks = 1
	}

	var mu sync.Mutex
	chunks := map[int][]byte{}
	persistent := map[int][]byte{}
	if req.PrefilledChunks > 0 {
		for i := 0; i < req.PrefilledChunks && i < totalChunks; i++ {
			start := i * client.ChunkSize
			end := start + client.ChunkSize
			if end > len(wire) {
				end = len(wire)
			}
			persistent[i] = append([]byte(nil), wire[start:end]...)
			chunks[i] = persistent[i]
		}
	}

	destPath := filepath.Join(dir, "dest.bin")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/upload/init":
			var body struct {
				FileHash    string `json:"file_hash"`
				TotalChunks int    `json:"total_chunks"`
				TotalSize   int64  `json:"total_size"`
				Compressed  bool   `json:"compressed"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSONErr(w, 400, "bad init")
				return
			}
			c := body.Compressed
			resp.InitCompressed = &c
			resp.WireTotalSize = body.TotalSize
			var received []int
			for idx := range persistent {
				received = append(received, idx)
			}
			sortInts(received)
			writeJSON(w, map[string]any{
				"upload_id":       body.FileHash,
				"received_chunks": received,
			})
		case "/api/files/upload/chunk":
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				writeJSONErr(w, 400, "bad form")
				return
			}
			idx, _ := strconv.Atoi(r.FormValue("chunk_index"))
			mu.Lock()
			resp.ChunkAttempts[idx]++
			resp.TotalChunkPosts++
			mu.Unlock()
			f, _, err := r.FormFile("chunk")
			if err != nil {
				writeJSONErr(w, 400, "missing chunk")
				return
			}
			data, _ := io.ReadAll(f)
			f.Close()
			mu.Lock()
			chunks[idx] = append([]byte(nil), data...)
			persistent[idx] = chunks[idx]
			mu.Unlock()
			writeJSON(w, map[string]any{"status": "ok", "chunk_index": idx})
		case "/api/files/upload/complete":
			resp.CompleteCalled = true
			var assembled bytes.Buffer
			for i := 0; i < totalChunks; i++ {
				assembled.Write(chunks[i])
			}
			raw := assembled.Bytes()
			if compressedExpected {
				gr, err := gzip.NewReader(bytes.NewReader(raw))
				if err != nil {
					writeJSONErr(w, 500, "gunzip: "+err.Error())
					return
				}
				var out bytes.Buffer
				if _, err := io.Copy(&out, gr); err != nil {
					gr.Close()
					writeJSONErr(w, 500, "gunzip copy: "+err.Error())
					return
				}
				gr.Close()
				raw = out.Bytes()
			}
			if !bytes.Equal(raw, content) {
				writeJSONErr(w, 500, "content mismatch")
				return
			}
			resp.AssembledRaw = append([]byte(nil), raw...)
			resp.ResultSize = int64(len(raw))
			writeJSON(w, map[string]any{"status": "ok", "path": destPath, "size": resp.ResultSize})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "")
	c.HTTPClient = srv.Client()
	result, err := c.UploadFile(localFile, destPath, client.UploadOptions{
		NoCompress: req.NoCompress,
		ChunkRetry: &client.ChunkRetryConfig{
			MaxAttempts: req.MaxChunkAttempts,
			Backoff:     func(int) int64 { return 0 },
		},
	}, nil)
	if err != nil {
		resp.UploadErr = err.Error()
		return resp, nil
	}
	if result != nil {
		resp.ResultSize = result.Size
	}
	t.Logf("evidence: compressed=%v wireSize=%d posts=%d attempts=%v resultSize=%d",
		resp.InitCompressed, resp.WireTotalSize, resp.TotalChunkPosts, resp.ChunkAttempts, resp.ResultSize)
	return resp, nil
}

func prepareWire(raw []byte, noCompress bool) ([]byte, bool, error) {
	if noCompress {
		return raw, false, nil
	}
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, false, err
	}
	zw.Header.ModTime = time.Time{}
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return nil, false, err
	}
	if err := zw.Close(); err != nil {
		return nil, false, err
	}
	gz := buf.Bytes()
	if len(gz) >= len(raw) {
		return raw, false, nil
	}
	return gz, true, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func sortInts(v []int) {
	for i := 0; i < len(v); i++ {
		for j := i + 1; j < len(v); j++ {
			if v[j] < v[i] {
				v[i], v[j] = v[j], v[i]
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```
