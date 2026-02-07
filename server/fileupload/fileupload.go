package fileupload

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// FileInfo represents information about a file on the server
type FileInfo struct {
	Exists   bool   `json:"exists"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	ModTime  string `json:"mod_time,omitempty"`
	IsDir    bool   `json:"is_dir"`
	FileMode string `json:"file_mode,omitempty"`
}

// RegisterAPI registers the file upload endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/files/check", handleCheck)
	mux.HandleFunc("/api/files/upload", handleUpload)
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Clean the path to prevent traversal
	cleanPath := filepath.Clean(req.Path)

	info, err := os.Stat(cleanPath)
	if os.IsNotExist(err) {
		writeJSON(w, FileInfo{
			Exists: false,
			Path:   cleanPath,
		})
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stat file: %v", err))
		return
	}

	writeJSON(w, FileInfo{
		Exists:   true,
		Path:     cleanPath,
		Size:     info.Size(),
		ModTime:  info.ModTime().Format(time.RFC3339),
		IsDir:    info.IsDir(),
		FileMode: info.Mode().String(),
	})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse form: %v", err))
		return
	}

	// Get the destination path
	destPath := r.FormValue("path")
	if destPath == "" {
		writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Clean the path
	destPath = filepath.Clean(destPath)

	// Get the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("file is required: %v", err))
		return
	}
	defer file.Close()

	// Ensure parent directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}

	// Write the file
	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create file: %v", err))
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write file: %v", err))
		return
	}

	writeJSON(w, map[string]any{
		"status":        "ok",
		"path":          destPath,
		"size":          written,
		"original_name": header.Filename,
	})
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
