# Client Upload Resilience Doctests

Unit tests for `client.Client.UploadFile` per-chunk retry on transient failures.

# DSN (Domain Specific Notion)

The upload resilience harness models chunked file transfer from CLI client to
ai-critic server file-upload API.

**Participants**

- **Client.UploadFile** — reads local file in 2 MiB chunks, calls init → chunk(s) → complete.
- **Chunk retry policy** — classifies errors as retryable; re-sends same chunk bytes with backoff.
- **Mock upload server** — `httptest.Server` implementing `/api/files/upload/{init,chunk,complete}`.
- **Attempt tracker** — counts POSTs per `chunk_index` to prove retries occurred.

**Behaviors**

- Transient 502/503/504 or transport errors trigger retry on the same `upload_id` + index.
- Non-retryable 4xx abort immediately without further attempts on that chunk.
- After retries exhausted, upload fails with last error.
- Server assembles chunks in order on complete; client verifies byte identity.

## Version

0.0.2

## Decision Tree

```
[UploadFile retry]
 |
 +-- transient-recovery/
 |    |
 |    +-- succeeds-after-502/                  (LEAF)  flaky chunk recovers; full file OK
 |    +-- succeeds-after-connection-reset/     (LEAF)  transport error then succeeds
 |
 +-- retry-exhaustion/
 |    |
 |    +-- aborts-after-max-attempts/   (LEAF)  always-502 until cap
 |
 +-- session-lost/
 |    |
 |    +-- recovers-after-mid-upload-404/  (LEAF)  server drops session; client re-inits
 |
 +-- cross-run-resume/
 |    |
 |    +-- skips-cached-chunks-on-reupload/ (LEAF)  same binary re-run skips known chunks
 |
 +-- non-retryable/
      |
      +-- fast-fail-on-400/             (LEAF)  400 → no retry
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `transient-recovery/succeeds-after-502` | Chunk fails twice with 502, succeeds on 3rd try; file intact |
| 2 | `transient-recovery/succeeds-after-connection-reset` | Transport reset twice on chunk 2, succeeds on 3rd try |
| 3 | `retry-exhaustion/aborts-after-max-attempts` | Permanent 502 exhausts attempts and errors |
| 4 | `session-lost/recovers-after-mid-upload-404` | Session dies after chunk 28; upload still completes |
| 5 | `cross-run-resume/skips-cached-chunks-on-reupload` | Chunks 0..N-2 cached; only missing chunk uploaded |
| 6 | `non-retryable/fast-fail-on-400` | HTTP 400 on chunk → single POST, immediate fail |

## How to Run

```sh
doctest vet ./client/tests/upload-resilience
doctest test ./client/tests/upload-resilience/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

type Request struct {
	TotalBytes         int64
	FlakyChunkIndex    int
	TransientFails     int
	FailStatus         int
	PermanentStatus    int
	MaxChunkAttempts   int
	AlwaysFailChunk    int // -1 disables; >=0 fails that chunk on every HTTP attempt
	TransportFailChunk     int
	TransportFailCount     int
	SessionDropAfterChunk int  // after storing this index, session becomes invalid
	PrefilledChunks       int // chunks 0..N-1 already on server before upload
}

type Response struct {
	UploadErr          string
	ResultPath         string
	ResultSize         int64
	ChunkAttempts      map[int]int
	TransportAttempts  map[int]int
	TotalChunkPosts    int
	CompleteCalled     bool
	InitCount          int
	UploadID           string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{
		ChunkAttempts:     make(map[int]int),
		TransportAttempts: make(map[int]int),
	}

	if req.TotalBytes <= 0 {
		req.TotalBytes = 5 * client.ChunkSize
	}
	if req.FailStatus == 0 {
		req.FailStatus = http.StatusBadGateway
	}
	if req.PermanentStatus == 0 {
		req.PermanentStatus = http.StatusBadRequest
	}
	if req.MaxChunkAttempts == 0 {
		req.MaxChunkAttempts = 5
	}

	localFile, wantContent := writeTempUploadFile(t, req.TotalBytes)
	destPath := "/tmp/upload-resilience-dest.bin"

	var mu sync.Mutex
	chunks := make(map[int][]byte)
	persistentChunks := make(map[int][]byte)
	var uploadID string
	sessionAlive := true
	chunkFailCounts := make(map[int]int)
	totalChunks := int((req.TotalBytes + client.ChunkSize - 1) / client.ChunkSize)

	if req.PrefilledChunks > 0 {
		for i := 0; i < req.PrefilledChunks && i < totalChunks; i++ {
			start := int64(i) * client.ChunkSize
			end := start + client.ChunkSize
			if end > req.TotalBytes {
				end = req.TotalBytes
			}
			persistentChunks[i] = append([]byte(nil), wantContent[start:end]...)
			chunks[i] = persistentChunks[i]
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/upload/init":
			mu.Lock()
			resp.InitCount++
			mu.Unlock()
			var body struct {
				Path        string `json:"path"`
				TotalChunks int    `json:"total_chunks"`
				TotalSize   int64  `json:"total_size"`
				FileHash    string `json:"file_hash"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSONErr(w, http.StatusBadRequest, "bad init")
				return
			}
			if body.FileHash != "" {
				uploadID = body.FileHash
			} else {
				uploadID = "test-upload-1"
			}
			sessionAlive = true
			var received []int
			for idx := range persistentChunks {
				received = append(received, idx)
			}
			sortInts(received)
			mu.Lock()
			resp.UploadID = uploadID
			mu.Unlock()
			writeJSON(w, map[string]any{
				"upload_id":       uploadID,
				"received_chunks": received,
			})

		case "/api/files/upload/chunk":
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				writeJSONErr(w, http.StatusBadRequest, "bad form")
				return
			}
			uid := r.FormValue("upload_id")
			idxStr := r.FormValue("chunk_index")
			idx, _ := strconv.Atoi(idxStr)

			mu.Lock()
			resp.ChunkAttempts[idx]++
			resp.TotalChunkPosts++
			mu.Unlock()

			hashMode := len(uid) == 64
			if !hashMode && (uid != uploadID || !sessionAlive) {
				writeJSONErr(w, http.StatusNotFound, "upload session not found")
				return
			}
			if hashMode && uid != uploadID {
				writeJSONErr(w, http.StatusNotFound, "upload session not found")
				return
			}

			if shouldFailChunk(req, idx, chunkFailCounts) {
				mu.Lock()
				chunkFailCounts[idx]++
				mu.Unlock()
				status := failStatus(req, idx)
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(status)
				fmt.Fprintf(w, "error code: %d", status)
				return
			}

			f, _, err := r.FormFile("chunk")
			if err != nil {
				writeJSONErr(w, http.StatusBadRequest, "missing chunk")
				return
			}
			data, _ := io.ReadAll(f)
			f.Close()
			mu.Lock()
			chunks[idx] = append([]byte(nil), data...)
			persistentChunks[idx] = chunks[idx]
			if req.SessionDropAfterChunk >= 0 && idx == req.SessionDropAfterChunk {
				sessionAlive = false
			}
			mu.Unlock()
			writeJSON(w, map[string]any{"status": "ok", "chunk_index": idx})

		case "/api/files/upload/complete":
			mu.Lock()
			resp.CompleteCalled = true
			mu.Unlock()
			var body struct {
				UploadID string `json:"upload_id"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			defer mu.Unlock()
			if len(chunks) != totalChunks {
				writeJSONErr(w, http.StatusBadRequest, fmt.Sprintf("only %d of %d chunks", len(chunks), totalChunks))
				return
			}
			var assembled bytes.Buffer
			for i := 0; i < totalChunks; i++ {
				assembled.Write(chunks[i])
			}
			if !bytes.Equal(assembled.Bytes(), wantContent) {
				writeJSONErr(w, http.StatusInternalServerError, "content mismatch")
				return
			}
			resp.ResultPath = destPath
			resp.ResultSize = int64(assembled.Len())
			writeJSON(w, map[string]any{"status": "ok", "path": destPath, "size": resp.ResultSize})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	transportTracker := &client.ChunkTransportTracker{Attempts: resp.TransportAttempts}
	c := client.New(srv.URL, "")
	base := srv.Client()
	base.Transport = client.WrapFlakyChunkTransport(base.Transport, client.FlakyChunkTransportConfig{
		FailChunkIndex: req.TransportFailChunk,
		FailCount:      req.TransportFailCount,
		Tracker:        transportTracker,
	})
	c.HTTPClient = base
	uploadOpts := client.UploadOptions{
		ChunkRetry: &client.ChunkRetryConfig{
			MaxAttempts: req.MaxChunkAttempts,
			Backoff:     func(int) int64 { return 0 },
		},
	}
	result, err := c.UploadFile(localFile, destPath, uploadOpts, nil)
	if err != nil {
		resp.UploadErr = err.Error()
		return resp, nil
	}
	if result != nil {
		resp.ResultPath = result.Path
		resp.ResultSize = result.Size
	}
	t.Logf("evidence: InitCount=%d TotalChunkPosts=%d ChunkAttempts=%v TransportAttempts=%v CompleteCalled=%v ResultSize=%d UploadErr=%q",
		resp.InitCount, resp.TotalChunkPosts, resp.ChunkAttempts, resp.TransportAttempts, resp.CompleteCalled, resp.ResultSize, resp.UploadErr)
	return resp, nil
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

func shouldFailChunk(req *Request, idx int, counts map[int]int) bool {
	if req.AlwaysFailChunk >= 0 && idx == req.AlwaysFailChunk {
		return true
	}
	if req.TransientFails > 0 && idx == req.FlakyChunkIndex {
		return counts[idx] < req.TransientFails
	}
	if req.FlakyChunkIndex >= 0 && req.PermanentStatus != 0 && req.TransientFails == 0 && req.AlwaysFailChunk < 0 && idx == req.FlakyChunkIndex {
		return true
	}
	return false
}

func failStatus(req *Request, idx int) int {
	if req.FlakyChunkIndex >= 0 && req.PermanentStatus != 0 && req.TransientFails == 0 && req.AlwaysFailChunk < 0 && idx == req.FlakyChunkIndex {
		return req.PermanentStatus
	}
	return req.FailStatus
}

func writeTempUploadFile(t *testing.T, size int64) (path string, content []byte) {
	t.Helper()
	content = make([]byte, size)
	for i := range content {
		content[i] = byte(i % 251)
	}
	dir := t.TempDir()
	path = filepath.Join(dir, "upload-src.bin")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path, content
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```