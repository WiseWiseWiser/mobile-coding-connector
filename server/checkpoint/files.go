package checkpoint

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileEntry represents a file or directory in the file browser.
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`  // relative to project root
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

// ListFiles lists entries in a directory within a project.
// path is relative to projectDir (empty string = root).
func ListFiles(projectDir string, relativePath string) ([]FileEntry, error) {
	absPath := projectDir
	if relativePath != "" {
		absPath = filepath.Join(projectDir, relativePath)
	}

	// Ensure we're within the project directory (prevent path traversal)
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, err
	}
	projAbs, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(absPath, projAbs) {
		return nil, os.ErrPermission
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var result []FileEntry
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files/dirs
		if strings.HasPrefix(name, ".") {
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

	entries, err := ListFiles(projectDir, path)
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
