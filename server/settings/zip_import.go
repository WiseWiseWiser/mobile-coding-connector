package settings

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// FileAction describes what will happen when a file is imported.
const (
	FileActionCreate    = "create"
	FileActionOverwrite = "overwrite"
	FileActionMerge     = "merge"
)

// Zip path prefixes used in the exported zip file.
// In the zip, ".ai-critic/" files are stored under "ai-critic/" (without the dot).
const (
	zipPrefixAICritic   = "ai-critic/"
	diskPrefixAICritic  = config.DataDir + "/"
	zipPrefixCloudflare = "cloudflare/"
)

// ImportFilePreview describes a single file in the import preview.
type ImportFilePreview struct {
	Path   string `json:"path"`   // path relative to zip root, e.g. "ai-critic/enc-key"
	Action string `json:"action"` // "create", "overwrite", or "merge"
	Size   int64  `json:"size"`   // size in bytes from the zip
}

// ImportZipPreview is the response from POST /api/settings/import-zip/preview
type ImportZipPreview struct {
	Files []ImportFilePreview `json:"files"`
}

func handleImportZipPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, _, err := readZipFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	preview := buildImportPreview(files)
	writeJSON(w, preview)
}

func handleImportZipConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, zipReader, err := readZipFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := applyZipImport(files, zipReader); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// readZipFromRequest reads the uploaded zip file from a multipart request.
// Returns the list of zip files, the zip reader, and any error.
func readZipFromRequest(r *http.Request) ([]*zip.File, *zip.Reader, error) {
	// Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		return nil, nil, fmt.Errorf("failed to parse form: %w", err)
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, nil, fmt.Errorf("missing 'file' field: %w", err)
	}
	defer file.Close()

	// Read all bytes into memory to create a zip reader
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid zip file: %w", err)
	}

	return zipReader.File, zipReader, nil
}

// buildImportPreview analyzes zip files and determines what action each file would trigger.
func buildImportPreview(files []*zip.File) *ImportZipPreview {
	var previews []ImportFilePreview

	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}

		zipPath := filepath.ToSlash(f.Name)
		action := determineFileAction(zipPath)
		previews = append(previews, ImportFilePreview{
			Path:   zipPath,
			Action: action,
			Size:   int64(f.UncompressedSize64),
		})
	}

	return &ImportZipPreview{Files: previews}
}

// determineFileAction figures out what will happen for a given zip path.
func determineFileAction(zipPath string) string {
	destPath := resolveDestPath(zipPath)
	if destPath == "" {
		return FileActionCreate
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return FileActionCreate
	}

	// Special merge logic for credentials (tokens are deduplicated)
	if zipPath == zipPrefixAICritic+"server-credentials" {
		return FileActionMerge
	}

	return FileActionOverwrite
}

// resolveDestPath maps a zip path to its actual filesystem destination.
// In the zip, "ai-critic/" maps to ".ai-critic/" on disk.
func resolveDestPath(zipPath string) string {
	// Files under ai-critic/ go to .ai-critic/ (relative to working directory)
	if strings.HasPrefix(zipPath, zipPrefixAICritic) {
		relName := strings.TrimPrefix(zipPath, zipPrefixAICritic)
		return diskPrefixAICritic + relName
	}

	// Files under cloudflare/ go to ~/.cloudflared/
	if strings.HasPrefix(zipPath, zipPrefixCloudflare) {
		name := strings.TrimPrefix(zipPath, zipPrefixCloudflare)
		dir := cloudflaredDir()
		if dir == "" {
			return ""
		}
		return filepath.Join(dir, name)
	}

	return ""
}

// applyZipImport extracts zip files to their destination paths.
func applyZipImport(files []*zip.File, _ *zip.Reader) error {
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}

		zipPath := filepath.ToSlash(f.Name)
		destPath := resolveDestPath(zipPath)
		if destPath == "" {
			continue
		}

		// Validate: prevent path traversal
		if strings.Contains(destPath, "..") {
			continue
		}

		content, err := readZipFile(f)
		if err != nil {
			return fmt.Errorf("read %s from zip: %w", zipPath, err)
		}

		// Special handling for server-credentials: merge tokens
		if zipPath == zipPrefixAICritic+"server-credentials" {
			if err := mergeCredentialsFile(destPath, content); err != nil {
				return fmt.Errorf("merge credentials: %w", err)
			}
			continue
		}

		// Determine file permissions
		perm := filePermission(zipPath)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, perm); err != nil {
			return fmt.Errorf("write %s: %w", destPath, err)
		}
	}
	return nil
}

// filePermission returns the appropriate file permission for a given zip path.
func filePermission(zipPath string) os.FileMode {
	// Private keys get restricted permissions
	if zipPath == zipPrefixAICritic+"enc-key" {
		return 0600
	}
	// Cloudflare files get restricted permissions
	if strings.HasPrefix(zipPath, zipPrefixCloudflare) {
		return 0600
	}
	return 0644
}

// mergeCredentialsFile merges new credential tokens with existing ones (deduplication).
func mergeCredentialsFile(destPath string, newContent []byte) error {
	existing := map[string]bool{}

	// Read existing tokens
	if data, err := os.ReadFile(destPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				existing[line] = true
			}
		}
	}

	// Add new tokens
	for _, line := range strings.Split(string(newContent), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			existing[line] = true
		}
	}

	// Write back all tokens
	var tokens []string
	for token := range existing {
		tokens = append(tokens, token)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	content := strings.Join(tokens, "\n") + "\n"
	return os.WriteFile(destPath, []byte(content), 0600)
}

// readZipFile reads the contents of a single file from a zip archive.
func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// handleImportZipBrowserData handles the browser-data portion of the import.
// Browser data (git configs, SSH keys) is stored in the zip as browser-data.json.
// The frontend reads this on import and saves to localStorage.
func handleImportZipBrowserData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, _, err := readZipFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Find and return browser-data.json contents
	for _, f := range files {
		if filepath.ToSlash(f.Name) == "browser-data.json" {
			content, err := readZipFile(f)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("read browser-data.json: %v", err))
				return
			}
			// Validate it's valid JSON
			var raw json.RawMessage
			if err := json.Unmarshal(content, &raw); err != nil {
				writeJSONError(w, http.StatusBadRequest, "browser-data.json is not valid JSON")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(content)
			return
		}
	}

	// No browser-data.json found
	writeJSON(w, map[string]any{})
}
