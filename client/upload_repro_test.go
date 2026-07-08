package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

// Phase-A prototypes for reported upload bugs (session 404 + full re-upload).

func TestRepro_sessionLost404_midUpload(t *testing.T) {
	const totalChunks = 10
	const dropAfter = 7 // chunks 0..7 ok; chunk 8 sees dead session

	localFile, _ := writeReproFile(t, int64(totalChunks*ChunkSize))
	dest := "/tmp/repro-dest.bin"

	var mu sync.Mutex
	chunks := make(map[int][]byte)
	uploadID := "sess-1"
	sessionAlive := true
	var totalPosts int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/upload/init":
			writeReproJSON(w, map[string]string{"upload_id": uploadID})
		case "/api/files/upload/chunk":
			mu.Lock()
			totalPosts++
			mu.Unlock()
			if !sessionAlive {
				writeReproJSONErr(w, http.StatusNotFound, "upload session not found")
				return
			}
			idx := parseReproChunkIndex(t, r)
			f, _, _ := r.FormFile("chunk")
			data, _ := io.ReadAll(f)
			f.Close()
			mu.Lock()
			chunks[idx] = data
			if idx == dropAfter {
				sessionAlive = false
			}
			mu.Unlock()
			writeReproJSON(w, map[string]any{"status": "ok"})
		case "/api/files/upload/complete":
			writeReproJSON(w, map[string]any{"status": "ok", "path": dest, "size": int64(len(chunks) * ChunkSize)})
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	c.HTTPClient = srv.Client()
	_, err := c.UploadFile(localFile, dest, UploadOptions{}, nil)
	if err == nil {
		t.Fatal("want error when server drops session mid-upload")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("upload session not found")) {
		t.Fatalf("err = %v", err)
	}
	t.Logf("confirmed: session lost after chunk %d, totalPosts=%d", dropAfter, totalPosts)
}

func TestRepro_secondRunReuploadsAllChunks(t *testing.T) {
	const totalChunks = 5
	localFile, _ := writeReproFile(t, int64(totalChunks*ChunkSize))
	dest := "/tmp/repro-dest2.bin"

	var mu sync.Mutex
	cached := make(map[int][]byte)
	var postsPerRun []int
	runPosts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/upload/init":
			mu.Lock()
			postsPerRun = append(postsPerRun, 0)
			runPosts = 0
			mu.Unlock()
			writeReproJSON(w, map[string]string{"upload_id": "sess"})
		case "/api/files/upload/chunk":
			idx := parseReproChunkIndex(t, r)
			f, _, _ := r.FormFile("chunk")
			data, _ := io.ReadAll(f)
			f.Close()
			mu.Lock()
			runPosts++
			// Server-side cache simulates prior partial upload still on disk.
			if _, ok := cached[idx]; !ok {
				cached[idx] = data
			}
			postsPerRun[len(postsPerRun)-1] = runPosts
			mu.Unlock()
			writeReproJSON(w, map[string]any{"status": "ok"})
		case "/api/files/upload/complete":
			writeReproJSON(w, map[string]any{"status": "ok", "path": dest, "size": int64(totalChunks * ChunkSize)})
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	c.HTTPClient = srv.Client()

	// Run 1: simulate failed upload after caching all chunks except last.
	for i := 0; i < totalChunks-1; i++ {
		cached[i] = []byte{byte(i)}
	}
	_, err := c.UploadFile(localFile, dest, UploadOptions{}, nil)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	firstPosts := postsPerRun[0]

	// Run 2: same file — user expects skip; client re-uploads everything today.
	_, err = c.UploadFile(localFile, dest, UploadOptions{}, nil)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	secondPosts := postsPerRun[1]
	t.Logf("firstRunPosts=%d secondRunPosts=%d cachedChunks=%d", firstPosts, secondPosts, len(cached))
	if secondPosts >= totalChunks {
		t.Fatalf("BUG: second run re-uploaded all %d chunks (want skip cached)", secondPosts)
	}
}

func writeReproFile(t *testing.T, size int64) (string, []byte) {
	t.Helper()
	data := make([]byte, size)
	dir := t.TempDir()
	path := filepath.Join(dir, "src.bin")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path, data
}

func parseReproChunkIndex(t *testing.T, r *http.Request) int {
	t.Helper()
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		t.Fatal(err)
	}
	idx, _ := strconv.Atoi(r.FormValue("chunk_index"))
	return idx
}

func writeReproJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeReproJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}