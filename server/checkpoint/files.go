package checkpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FileEntry represents a file or directory in the file browser.
type FileEntry struct {
	Name         string `json:"name"`
	Path         string `json:"path"` // relative to project root
	IsDir        bool   `json:"is_dir"`
	Size         int64  `json:"size,omitempty"`
	ModifiedTime string `json:"modified_time,omitempty"`
}

// ListFiles lists entries in a directory within a project.
// Returns a slice of FileEntry and an error if the directory cannot be read.
// If showHidden is true, hidden files (dot files) will be included in the result.
func ListFiles(projectDir string, relativePath string, showHidden bool) ([]FileEntry, error) {
	// Clean and validate the path
	absPath := filepath.Join(projectDir, relativePath)

	// Security: ensure the path is within projectDir
	if !strings.HasPrefix(absPath, projectDir) {
		return nil, fmt.Errorf("path %q is outside project directory", relativePath)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var result []FileEntry
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files/dirs unless showHidden is true
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		// Skip node_modules
		if name == "node_modules" {
			continue
		}

		entryPath := name
		if relativePath != "" {
			entryPath = relativePath + "/" + name
		}

		fe := FileEntry{
			Name:  name,
			Path:  entryPath,
			IsDir: entry.IsDir(),
		}

		if !entry.IsDir() {
			info, infoErr := entry.Info()
			if infoErr == nil {
				fe.Size = info.Size()
			}
		}

		result = append(result, fe)
	}

	// Sort: directories first, then files, alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// ReadFileContent reads a file from the project directory (for viewing).
func ReadProjectFile(projectDir, relativePath string) (string, error) {
	absPath := filepath.Join(projectDir, relativePath)

	// Security check
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", err
	}
	projAbs, err := filepath.Abs(projectDir)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, projAbs) {
		return "", os.ErrPermission
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// handleListFiles handles GET /api/files
func handleListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		respondErr(w, http.StatusBadRequest, "project_dir is required")
		return
	}

	path := r.URL.Query().Get("path")

	// Check if hidden files should be shown
	showHidden := r.URL.Query().Get("hidden") == "true"

	entries, err := ListFiles(projectDir, path, showHidden)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, entries)
}

// handleReadFile handles GET /api/files/content
func handleReadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		respondErr(w, http.StatusBadRequest, "project_dir is required")
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		respondErr(w, http.StatusBadRequest, "path is required")
		return
	}

	content, err := ReadProjectFile(projectDir, filePath)
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"content": content, "path": filePath})
}

// handleHomeDir handles GET /api/files/home
func handleHomeDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"home_dir": home})
}

// ListServerFiles lists entries in a server directory, allowing navigation to root.
// basePath is the starting directory (e.g., home), relativePath navigates from there.
// Shows hidden files and allows going up to root.
func ListServerFiles(basePath string, relativePath string) ([]FileEntry, error) {
	// Calculate absolute path
	absPath := basePath
	if relativePath != "" {
		absPath = filepath.Join(basePath, relativePath)
	}

	// Clean and resolve the path
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, err
	}

	// Ensure we don't go below root
	if absPath == "" || absPath == "." {
		absPath = "/"
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var result []FileEntry
	for _, entry := range entries {
		name := entry.Name()

		// Show all files including hidden ones (dot files)
		// Only skip . and .. which are handled separately
		if name == "." || name == ".." {
			continue
		}

		entryPath := name
		if relativePath != "" && relativePath != "." {
			entryPath = relativePath + "/" + name
		}

		fe := FileEntry{
			Name:  name,
			Path:  entryPath,
			IsDir: entry.IsDir(),
		}

		info, infoErr := entry.Info()
		if infoErr == nil {
			fe.ModifiedTime = info.ModTime().Format(time.RFC3339)
			if !entry.IsDir() {
				fe.Size = info.Size()
			}
		}

		result = append(result, fe)
	}

	// Sort: directories first, then files, alphabetically
	// Hidden files (starting with .) come after visible files
	sort.Slice(result, func(i, j int) bool {
		iHidden := strings.HasPrefix(result[i].Name, ".")
		jHidden := strings.HasPrefix(result[j].Name, ".")

		// First sort by visibility (non-hidden first)
		if iHidden != jHidden {
			return !iHidden
		}

		// Then sort directories before files
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}

		// Finally alphabetical
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// handleListServerFiles handles GET /api/server/files for server-wide file browsing
func handleListServerFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	basePath := r.URL.Query().Get("base_path")
	if basePath == "" {
		respondErr(w, http.StatusBadRequest, "base_path is required")
		return
	}

	path := r.URL.Query().Get("path")

	entries, err := ListServerFiles(basePath, path)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, entries)
}

// FilePartialContent represents partial file content with pagination info
type FilePartialContent struct {
	Content   string `json:"content"`
	TotalSize int64  `json:"totalSize"`
	Offset    int64  `json:"offset"`
	HasMore   bool   `json:"hasMore"`
}

// ReadFilePartial reads a portion of a file
func ReadFilePartial(filePath string, offset int64, limit int64) (*FilePartialContent, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Seek to offset
	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	// Read up to limit bytes
	buf := make([]byte, limit)
	n, err := file.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return &FilePartialContent{
		Content:   string(buf[:n]),
		TotalSize: info.Size(),
		Offset:    offset,
		HasMore:   offset+int64(n) < info.Size(),
	}, nil
}

// handleServerFileContent handles GET/POST /api/server/files/content
func handleServerFileContent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filePath := r.URL.Query().Get("path")
		if filePath == "" {
			respondErr(w, http.StatusBadRequest, "path is required")
			return
		}

		offsetStr := r.URL.Query().Get("offset")
		limitStr := r.URL.Query().Get("limit")

		var offset, limit int64 = 0, 0 // 0 means read all
		if offsetStr != "" {
			offset, _ = strconv.ParseInt(offsetStr, 10, 64)
		}
		if limitStr != "" {
			limit, _ = strconv.ParseInt(limitStr, 10, 64)
		}

		var result interface{}
		var err error

		if limit > 0 {
			result, err = ReadFilePartial(filePath, offset, limit)
		} else {
			content, err := os.ReadFile(filePath)
			if err != nil {
				respondErr(w, http.StatusNotFound, err.Error())
				return
			}
			result = map[string]string{"content": string(content)}
		}

		if err != nil {
			respondErr(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var req struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Path == "" {
			respondErr(w, http.StatusBadRequest, "path is required")
			return
		}

		err := os.WriteFile(req.Path, []byte(req.Content), 0644)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
