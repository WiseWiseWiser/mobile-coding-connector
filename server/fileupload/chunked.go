package fileupload

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

// chunkSession tracks an in-progress chunked upload.
type chunkSession struct {
	ID          string
	DestPath    string
	TotalChunks int
	TotalSize   int64
	ChmodExec   bool
	TempDir     string
	CreatedAt   time.Time
	Received    map[int]bool // chunk index -> received
}

var (
	sessionMu sync.Mutex
	sessions  = map[string]*chunkSession{}
)

func init() {
	// Periodically clean up stale sessions (older than 30 minutes)
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			cleanupStaleSessions(30 * time.Minute)
		}
	}()
}

func cleanupStaleSessions(maxAge time.Duration) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, s := range sessions {
		if s.CreatedAt.Before(cutoff) {
			os.RemoveAll(s.TempDir)
			delete(sessions, id)
		}
	}
}

func generateUploadID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// handleUploadInit starts a new chunked upload session.
// POST /api/files/upload/init
// Body: { "path": "/dest/path", "total_chunks": 5, "total_size": 10485760 }
func handleUploadInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path        string `json:"path"`
		TotalChunks int    `json:"total_chunks"`
		TotalSize   int64  `json:"total_size"`
		ChmodExec   bool   `json:"chmod_exec"`
		FileHash    string `json:"file_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}
	if req.TotalChunks <= 0 {
		writeJSONError(w, http.StatusBadRequest, "total_chunks must be positive")
		return
	}

	destPath := filepath.Clean(req.Path)

	if req.FileHash != "" {
		if !isFileHash(req.FileHash) {
			writeJSONError(w, http.StatusBadRequest, "invalid file_hash")
			return
		}
		dir, err := uploadCacheDir(req.FileHash)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to prepare cache dir: %v", err))
			return
		}
		if err := saveUploadMeta(dir, uploadMeta{
			DestPath:    destPath,
			TotalChunks: req.TotalChunks,
			TotalSize:   req.TotalSize,
			ChmodExec:   req.ChmodExec,
		}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to save upload meta: %v", err))
			return
		}
		received, err := listCachedChunkIndices(dir)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list cached chunks: %v", err))
			return
		}
		writeJSON(w, map[string]any{
			"upload_id":       req.FileHash,
			"received_chunks": received,
		})
		return
	}

	// Create temp directory for chunks
	tempDir, err := os.MkdirTemp("", "upload-chunks-*")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create temp dir: %v", err))
		return
	}

	id := generateUploadID()
	session := &chunkSession{
		ID:          id,
		DestPath:    destPath,
		TotalChunks: req.TotalChunks,
		TotalSize:   req.TotalSize,
		ChmodExec:   req.ChmodExec,
		TempDir:     tempDir,
		CreatedAt:   time.Now(),
		Received:    make(map[int]bool),
	}

	sessionMu.Lock()
	sessions[id] = session
	sessionMu.Unlock()

	writeJSON(w, map[string]string{
		"upload_id": id,
	})
}

// handleUploadChunk receives a single chunk.
// POST /api/files/upload/chunk (multipart form: upload_id, chunk_index, chunk)
func handleUploadChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 4MB per chunk)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
		return
	}

	uploadID := r.FormValue("upload_id")
	chunkIndexStr := r.FormValue("chunk_index")

	if uploadID == "" || chunkIndexStr == "" {
		writeJSONError(w, http.StatusBadRequest, "upload_id and chunk_index are required")
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid chunk_index")
		return
	}

	if isFileHash(uploadID) {
		handleHashUploadChunk(w, uploadID, chunkIndex, r)
		return
	}

	sessionMu.Lock()
	session, ok := sessions[uploadID]
	sessionMu.Unlock()
	if !ok {
		writeJSONError(w, http.StatusNotFound, "upload session not found")
		return
	}

	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("chunk_index out of range [0, %d)", session.TotalChunks))
		return
	}

	// Get the chunk file
	chunkFile, _, err := r.FormFile("chunk")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("chunk file is required: %v", err))
		return
	}
	defer chunkFile.Close()

	// Save chunk to temp dir
	chunkPath := filepath.Join(session.TempDir, fmt.Sprintf("chunk_%05d", chunkIndex))
	dst, err := os.Create(chunkPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create chunk file: %v", err))
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, chunkFile)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write chunk: %v", err))
		return
	}

	sessionMu.Lock()
	session.Received[chunkIndex] = true
	receivedCount := len(session.Received)
	sessionMu.Unlock()

	writeJSON(w, map[string]any{
		"status":         "ok",
		"chunk_index":    chunkIndex,
		"chunk_size":     written,
		"received_count": receivedCount,
		"total_chunks":   session.TotalChunks,
	})
}

// handleUploadComplete combines all chunks into the final file.
// POST /api/files/upload/complete
// Body: { "upload_id": "..." }
func handleUploadComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if isFileHash(req.UploadID) {
		handleHashUploadComplete(w, req.UploadID)
		return
	}

	sessionMu.Lock()
	session, ok := sessions[req.UploadID]
	if ok {
		delete(sessions, req.UploadID) // Remove session so it can't be completed twice
	}
	sessionMu.Unlock()

	if !ok {
		writeJSONError(w, http.StatusNotFound, "upload session not found")
		return
	}

	// Verify all chunks received
	if len(session.Received) != session.TotalChunks {
		os.RemoveAll(session.TempDir)
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("only %d of %d chunks received", len(session.Received), session.TotalChunks))
		return
	}

	// Ensure parent directory exists
	dir := filepath.Dir(session.DestPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		os.RemoveAll(session.TempDir)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}

	// Combine chunks in order
	dst, err := os.OpenFile(session.DestPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		os.RemoveAll(session.TempDir)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create destination file: %v", err))
		return
	}
	defer dst.Close()

	// List and sort chunk files
	chunkFiles, err := filepath.Glob(filepath.Join(session.TempDir, "chunk_*"))
	if err != nil {
		os.RemoveAll(session.TempDir)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list chunks: %v", err))
		return
	}
	sort.Strings(chunkFiles) // chunk_00000, chunk_00001, ... sorts correctly

	var totalWritten int64
	for _, chunkPath := range chunkFiles {
		src, err := os.Open(chunkPath)
		if err != nil {
			os.RemoveAll(session.TempDir)
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read chunk: %v", err))
			return
		}
		n, err := io.Copy(dst, src)
		src.Close()
		if err != nil {
			os.RemoveAll(session.TempDir)
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write chunk to destination: %v", err))
			return
		}
		totalWritten += n
	}

	// Cleanup temp directory
	os.RemoveAll(session.TempDir)

	if session.ChmodExec {
		if err := os.Chmod(session.DestPath, 0755); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to chmod destination file: %v", err))
			return
		}
	}

	// Return absolute path so clients see the real final location.
	absPath, absErr := filepath.Abs(session.DestPath)
	if absErr != nil {
		absPath = session.DestPath
	}

	writeJSON(w, map[string]any{
		"status": "ok",
		"path":   absPath,
		"size":   totalWritten,
	})
}

func handleHashUploadChunk(w http.ResponseWriter, uploadID string, chunkIndex int, r *http.Request) {
	dir, err := uploadCacheDir(uploadID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "upload session not found")
		return
	}
	meta, err := loadUploadMeta(dir)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "upload session not found")
		return
	}
	if chunkIndex < 0 || chunkIndex >= meta.TotalChunks {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("chunk_index out of range [0, %d)", meta.TotalChunks))
		return
	}
	chunkFile, _, err := r.FormFile("chunk")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("chunk file is required: %v", err))
		return
	}
	defer chunkFile.Close()
	data, err := io.ReadAll(chunkFile)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read chunk: %v", err))
		return
	}
	if path, ok := findCachedChunk(dir, chunkIndex); ok && chunkMatchesHash(path, data) {
		writeJSON(w, map[string]any{
			"status":       "ok",
			"chunk_index":  chunkIndex,
			"chunk_size":   len(data),
			"skipped":      true,
			"total_chunks": meta.TotalChunks,
		})
		return
	}
	if _, err := saveCachedChunk(dir, chunkIndex, data); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write chunk: %v", err))
		return
	}
	received, _ := listCachedChunkIndices(dir)
	writeJSON(w, map[string]any{
		"status":         "ok",
		"chunk_index":    chunkIndex,
		"chunk_size":     len(data),
		"received_count": len(received),
		"total_chunks":   meta.TotalChunks,
	})
}

func handleHashUploadComplete(w http.ResponseWriter, uploadID string) {
	dir, err := uploadCacheDir(uploadID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "upload session not found")
		return
	}
	meta, err := loadUploadMeta(dir)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "upload session not found")
		return
	}
	received, err := listCachedChunkIndices(dir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list chunks: %v", err))
		return
	}
	if len(received) != meta.TotalChunks {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("only %d of %d chunks received", len(received), meta.TotalChunks))
		return
	}
	parent := filepath.Dir(meta.DestPath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}
	totalWritten, err := assembleCachedFile(dir, meta, meta.DestPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to assemble file: %v", err))
		return
	}
	if meta.ChmodExec {
		if err := os.Chmod(meta.DestPath, 0755); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to chmod destination file: %v", err))
			return
		}
	}
	removeUploadCache(dir)
	absPath, absErr := filepath.Abs(meta.DestPath)
	if absErr != nil {
		absPath = meta.DestPath
	}
	writeJSON(w, map[string]any{
		"status": "ok",
		"path":   absPath,
		"size":   totalWritten,
	})
}
