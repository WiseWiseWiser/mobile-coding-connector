package filetransfer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

const (
	maxUploadSize  = 128 << 20 // 128 MB
	maxScratchSize = 256 << 10 // 256 KB
	scratchFile    = "scratch.json"
)

// FileTransferEntry describes one file in the transfer inbox.
type FileTransferEntry struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploaded_at"`
}

// ScratchEntry is the shared scratch pad blob.
type ScratchEntry struct {
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

// RegisterAPI registers dedicated file-transfer inbox endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/file-transfer", handleRoot)
	mux.HandleFunc("/api/file-transfer/upload", handleUpload)
	mux.HandleFunc("/api/file-transfer/download", handleDownload)
	mux.HandleFunc("/api/file-transfer/scratch", handleScratch)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleList(w, r)
	case http.MethodDelete:
		handleDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func ensureDir() (string, error) {
	dir := config.FileTransferDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create file-transfer dir: %w", err)
	}
	return dir, nil
}

func handleList(w http.ResponseWriter, r *http.Request) {
	dir, err := ensureDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("read file-transfer dir: %v", err))
		return
	}

	files := make([]FileTransferEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == scratchFile {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileTransferEntry{
			Name:       entry.Name(),
			Size:       info.Size(),
			UploadedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].UploadedAt > files[j].UploadedAt
	})

	writeJSON(w, map[string]any{"files": files})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("file is required: %v", err))
		return
	}
	defer file.Close()

	dir, err := ensureDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	storedName, err := dedupFilename(dir, header.Filename)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	dstPath := filepath.Join(dir, storedName)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create file: %v", err))
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(dstPath)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write file: %v", err))
		return
	}

	info, err := os.Stat(dstPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stat file: %v", err))
		return
	}

	writeJSON(w, map[string]any{
		"id":          storedName,
		"name":        storedName,
		"size":        written,
		"uploaded_at": info.ModTime().UTC().Format(time.RFC3339),
	})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name, err := safeFilename(r.URL.Query().Get("name"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	dir, err := ensureDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filePath := filepath.Join(dir, name)
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusNotFound, "file not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stat file: %v", err))
		return
	}
	if info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "not a file")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	http.ServeFile(w, r, filePath)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name, err := safeFilename(req.Name)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	dir, err := ensureDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filePath := filepath.Join(dir, name)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusNotFound, "file not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete file: %v", err))
		return
	}

	writeJSON(w, map[string]bool{"ok": true})
}

func safeFilename(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	base := filepath.Base(filepath.Clean(name))
	if base == "." || base == ".." {
		return "", fmt.Errorf("invalid filename")
	}
	return base, nil
}

func dedupFilename(dir, originalName string) (string, error) {
	base, err := safeFilename(filepath.Base(originalName))
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(filepath.Join(dir, base)); os.IsNotExist(err) {
		return base, nil
	}

	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", nameWithoutExt, i, ext)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate, nil
		}
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func scratchPath() string {
	return filepath.Join(config.FileTransferDir, scratchFile)
}

func readScratch() (ScratchEntry, error) {
	path := scratchPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ScratchEntry{Content: "", UpdatedAt: ""}, nil
		}
		return ScratchEntry{}, fmt.Errorf("read scratch: %w", err)
	}
	var entry ScratchEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return ScratchEntry{}, fmt.Errorf("parse scratch: %w", err)
	}
	return entry, nil
}

func handleScratch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleScratchGet(w, r)
	case http.MethodPut:
		handleScratchPut(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleScratchGet(w http.ResponseWriter, r *http.Request) {
	entry, err := readScratch()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, entry)
}

func handleScratchPut(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxScratchSize+1024)
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Content) > maxScratchSize {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("content exceeds maximum size of %d bytes", maxScratchSize))
		return
	}

	if _, err := ensureDir(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	entry := ScratchEntry{
		Content:   req.Content,
		UpdatedAt: updatedAt,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("marshal scratch: %v", err))
		return
	}
	if err := os.WriteFile(scratchPath(), data, 0o644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("write scratch: %v", err))
		return
	}
	writeJSON(w, entry)
}