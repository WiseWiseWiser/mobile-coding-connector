package checkpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
)

// FileSnapshot stores the state of a single file at a checkpoint (metadata only).
type FileSnapshot struct {
	Path   string `json:"path"`
	Status string `json:"status"` // "added", "modified", "deleted"
}

// CheckpointMeta is the metadata stored in checkpoint.json.
type CheckpointMeta struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Message   string         `json:"message,omitempty"`
	Timestamp string         `json:"timestamp"`
	Files     []FileSnapshot `json:"files"`
}

// Checkpoint is a named snapshot of changed files at a point in time.
// Content is loaded lazily from files/ directory.
type Checkpoint struct {
	CheckpointMeta
	DirPath string `json:"-"` // path to checkpoint directory
}

// CheckpointSummary is the lightweight listing version (without file contents).
type CheckpointSummary struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
	FileCount int    `json:"file_count"`
}

// ChangedFile represents a file that has changed (without content).
type ChangedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"` // "added", "modified", "deleted"
}

// --- Storage ---
// New structure: .ai-critic/projects/{project}/checkpoints/{index}_{name}/
//   - checkpoint.json: metadata
//   - files/: directory containing file contents (not for deleted files)

var baseDir = config.ProjectsDir

var mu sync.RWMutex

// projectCheckpointsDir returns the checkpoints directory for a project.
func projectCheckpointsDir(projectName string) string {
	return filepath.Join(baseDir, projectName, "checkpoints")
}

// checkpointDirName creates a directory name like "0_checkpoint_name".
func checkpointDirName(index int, name string) string {
	// Sanitize name for filesystem
	safeName := sanitizeName(name)
	return fmt.Sprintf("%d_%s", index, safeName)
}

// sanitizeName makes a name safe for use as a directory name.
func sanitizeName(name string) string {
	// Replace spaces with underscores, remove special chars
	name = strings.ReplaceAll(name, " ", "_")
	// Keep only alphanumeric, underscore, hyphen
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	name = reg.ReplaceAllString(name, "")
	if name == "" {
		name = "checkpoint"
	}
	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}
	return name
}

// parseCheckpointDirName extracts index from directory name like "0_checkpoint_name".
func parseCheckpointDirName(dirName string) (index int, name string, ok bool) {
	idx := strings.Index(dirName, "_")
	if idx < 0 {
		return 0, "", false
	}
	indexStr := dirName[:idx]
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return 0, "", false
	}
	name = dirName[idx+1:]
	return index, name, true
}

func ensureProjectDir(projectName string) error {
	return os.MkdirAll(projectCheckpointsDir(projectName), 0755)
}

// loadCheckpointMeta loads the checkpoint.json from a checkpoint directory.
func loadCheckpointMeta(cpDir string) (*CheckpointMeta, error) {
	data, err := os.ReadFile(filepath.Join(cpDir, "checkpoint.json"))
	if err != nil {
		return nil, err
	}
	var meta CheckpointMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// saveCheckpointMeta saves the checkpoint.json to a checkpoint directory.
func saveCheckpointMeta(cpDir string, meta *CheckpointMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cpDir, "checkpoint.json"), data, 0644)
}

// loadCheckpoints loads all checkpoints for a project (metadata only).
func loadCheckpoints(projectName string) ([]Checkpoint, error) {
	cpBaseDir := projectCheckpointsDir(projectName)
	entries, err := os.ReadDir(cpBaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Checkpoint{}, nil
		}
		return nil, err
	}

	var checkpoints []Checkpoint
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		index, _, ok := parseCheckpointDirName(entry.Name())
		if !ok {
			continue
		}
		cpDir := filepath.Join(cpBaseDir, entry.Name())
		meta, err := loadCheckpointMeta(cpDir)
		if err != nil {
			continue // skip invalid checkpoints
		}
		checkpoints = append(checkpoints, Checkpoint{
			CheckpointMeta: *meta,
			DirPath:        cpDir,
		})
		// Ensure ID matches index for consistency
		checkpoints[len(checkpoints)-1].ID = index
	}

	// Sort by ID
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].ID < checkpoints[j].ID
	})

	return checkpoints, nil
}

// getFileContent reads file content from the checkpoint's files/ directory.
func getFileContent(cpDir, filePath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(cpDir, "files", filePath))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getOriginalContent reads original (git HEAD) content from the checkpoint's original/ directory.
func getOriginalContent(cpDir, filePath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(cpDir, "original", filePath))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// saveFileContent saves file content to the checkpoint's files/ directory.
func saveFileContent(cpDir, filePath, content string) error {
	fullPath := filepath.Join(cpDir, "files", filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// saveOriginalContent saves original (git HEAD) content to the checkpoint's original/ directory.
func saveOriginalContent(cpDir, filePath, content string) error {
	fullPath := filepath.Join(cpDir, "original", filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// --- Git helpers ---

// gitChangedFiles returns list of changed files in the working tree compared to HEAD.
// Uses git diff --name-status.
func gitChangedFiles(projectDir string) ([]ChangedFile, error) {
	out, err := gitrunner.Diff("--name-status", "HEAD").Dir(projectDir).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	// Also include untracked files
	untrackedOut, err := gitrunner.LsFiles("--others", "--exclude-standard").Dir(projectDir).Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	files := make([]ChangedFile, 0)
	seen := map[string]bool{}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		status := parseGitStatus(parts[0])
		path := parts[1]
		files = append(files, ChangedFile{Path: path, Status: status})
		seen[path] = true
	}

	for _, line := range strings.Split(strings.TrimSpace(string(untrackedOut)), "\n") {
		if line == "" {
			continue
		}
		if seen[line] {
			continue
		}
		files = append(files, ChangedFile{Path: line, Status: "added"})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func parseGitStatus(s string) string {
	switch {
	case strings.HasPrefix(s, "A"):
		return "added"
	case strings.HasPrefix(s, "D"):
		return "deleted"
	default:
		return "modified"
	}
}

func readFileContent(projectDir, path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, path))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func gitFileContent(projectDir, path string) (string, error) {
	out, err := gitrunner.Show("HEAD:" + path).Dir(projectDir).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// --- Core operations ---

// ListCheckpoints returns summaries for all checkpoints of a project.
func ListCheckpoints(projectName string) ([]CheckpointSummary, error) {
	mu.RLock()
	defer mu.RUnlock()

	list, err := loadCheckpoints(projectName)
	if err != nil {
		return nil, err
	}

	summaries := make([]CheckpointSummary, len(list))
	for i, cp := range list {
		summaries[i] = CheckpointSummary{
			ID:        cp.ID,
			Name:      cp.Name,
			Message:   cp.Message,
			Timestamp: cp.Timestamp,
			FileCount: len(cp.Files),
		}
	}
	return summaries, nil
}

// GetCheckpoint returns the checkpoint metadata.
func GetCheckpoint(projectName string, id int) (*Checkpoint, error) {
	mu.RLock()
	defer mu.RUnlock()

	list, err := loadCheckpoints(projectName)
	if err != nil {
		return nil, err
	}

	for _, cp := range list {
		if cp.ID == id {
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("checkpoint %d not found", id)
}

// GetCheckpointFileContent reads the content of a file from a checkpoint.
func GetCheckpointFileContent(projectName string, id int, filePath string) (string, error) {
	mu.RLock()
	defer mu.RUnlock()

	list, err := loadCheckpoints(projectName)
	if err != nil {
		return "", err
	}

	for _, cp := range list {
		if cp.ID == id {
			return getFileContent(cp.DirPath, filePath)
		}
	}
	return "", fmt.Errorf("checkpoint %d not found", id)
}

// CreateCheckpointRequest is the request body for creating a checkpoint.
type CreateCheckpointRequest struct {
	ProjectDir string   `json:"project_dir"` // absolute path to project
	Name       string   `json:"name"`        // optional user-provided name
	Message    string   `json:"message"`     // optional user-provided message
	FilePaths  []string `json:"file_paths"`  // list of files to include (from changed files)
}

// CreateCheckpoint creates a new checkpoint from the specified changed files.
func CreateCheckpoint(projectName string, req CreateCheckpointRequest) (*CheckpointSummary, error) {
	mu.Lock()
	defer mu.Unlock()

	list, err := loadCheckpoints(projectName)
	if err != nil {
		return nil, err
	}

	// Determine next index (1-based)
	nextIndex := 1
	if len(list) > 0 {
		nextIndex = list[len(list)-1].ID + 1
	}

	name := req.Name
	if name == "" {
		name = fmt.Sprintf("Checkpoint #%d", nextIndex)
	}

	// Create checkpoint directory
	if err := ensureProjectDir(projectName); err != nil {
		return nil, err
	}
	cpDirName := checkpointDirName(nextIndex, name)
	cpDir := filepath.Join(projectCheckpointsDir(projectName), cpDirName)
	if err := os.MkdirAll(cpDir, 0755); err != nil {
		return nil, err
	}

	// Build file snapshots and save file contents
	var files []FileSnapshot
	for _, path := range req.FilePaths {
		status := "modified"

		// Read current content from disk
		diskContent, diskErr := readFileContent(req.ProjectDir, path)

		// Check git HEAD content to determine status
		gitContent, gitErr := gitFileContent(req.ProjectDir, path)

		if gitErr != nil {
			// File doesn't exist in HEAD -> added
			status = "added"
		}
		if diskErr != nil {
			// File doesn't exist on disk -> deleted
			status = "deleted"
		} else {
			// Save file content to files/ directory
			if err := saveFileContent(cpDir, path, diskContent); err != nil {
				// Cleanup on error
				os.RemoveAll(cpDir)
				return nil, fmt.Errorf("failed to save file %s: %w", path, err)
			}
		}

		// For modified or deleted files, save the original (git HEAD) content
		if status == "modified" || status == "deleted" {
			if err := saveOriginalContent(cpDir, path, gitContent); err != nil {
				os.RemoveAll(cpDir)
				return nil, fmt.Errorf("failed to save original %s: %w", path, err)
			}
		}

		files = append(files, FileSnapshot{
			Path:   path,
			Status: status,
		})
	}

	meta := &CheckpointMeta{
		ID:        nextIndex,
		Name:      name,
		Message:   req.Message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Files:     files,
	}

	if err := saveCheckpointMeta(cpDir, meta); err != nil {
		os.RemoveAll(cpDir)
		return nil, err
	}

	summary := &CheckpointSummary{
		ID:        meta.ID,
		Name:      meta.Name,
		Message:   meta.Message,
		Timestamp: meta.Timestamp,
		FileCount: len(meta.Files),
	}
	return summary, nil
}

// DeleteCheckpoint removes a checkpoint by ID.
func DeleteCheckpoint(projectName string, id int) error {
	mu.Lock()
	defer mu.Unlock()

	list, err := loadCheckpoints(projectName)
	if err != nil {
		return err
	}

	for _, cp := range list {
		if cp.ID == id {
			// Remove the checkpoint directory
			return os.RemoveAll(cp.DirPath)
		}
	}
	return fmt.Errorf("checkpoint %d not found", id)
}

// GetCurrentChanges returns the list of files changed compared to the last checkpoint
// (or git HEAD if no checkpoints exist).
func GetCurrentChanges(projectName, projectDir string) ([]ChangedFile, error) {
	mu.RLock()
	defer mu.RUnlock()

	// Get files changed since git HEAD
	gitFiles, err := gitChangedFiles(projectDir)
	if err != nil {
		return nil, err
	}

	// Load the latest checkpoint to filter out already-checkpointed files
	list, err := loadCheckpoints(projectName)
	if err != nil || len(list) == 0 {
		return gitFiles, nil
	}

	latestCP := list[len(list)-1]

	// Build set of files in the latest checkpoint
	cpFileSet := make(map[string]string) // path -> status
	for _, f := range latestCP.Files {
		cpFileSet[f.Path] = f.Status
	}

	// Filter: keep only files whose current content differs from the checkpoint
	var result []ChangedFile
	for _, cf := range gitFiles {
		cpStatus, inCheckpoint := cpFileSet[cf.Path]
		if !inCheckpoint {
			// File wasn't in the last checkpoint, always show it
			result = append(result, cf)
			continue
		}

		// File was in the checkpoint — compare current content to checkpoint content
		if cf.Status == "deleted" {
			// File is deleted now; if it was also deleted in checkpoint, skip it
			if cpStatus == "deleted" {
				continue
			}
			result = append(result, cf)
			continue
		}

		// Read current disk content
		currentContent, err := readFileContent(projectDir, cf.Path)
		if err != nil {
			// Can't read file, show it as changed
			result = append(result, cf)
			continue
		}

		// Read checkpoint's saved content
		cpContent, err := getFileContent(latestCP.DirPath, cf.Path)
		if err != nil {
			// Can't read checkpoint file (e.g. was deleted in checkpoint), show it
			result = append(result, cf)
			continue
		}

		// If contents differ, the file has changed since the checkpoint
		if currentContent != cpContent {
			result = append(result, cf)
		}
		// Otherwise skip — file hasn't changed since checkpoint
	}

	return result, nil
}

// --- HTTP API ---

// RegisterAPI registers the checkpoint and file browser API endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/checkpoints", handleCheckpoints)
	mux.HandleFunc("/api/checkpoints/", handleCheckpointByID)
	mux.HandleFunc("/api/checkpoints/diff", handleCurrentDiff)
	mux.HandleFunc("/api/checkpoints/diff/file", handleSingleFileDiff)
	mux.HandleFunc("/api/files", handleListFiles)
	mux.HandleFunc("/api/files/content", handleReadFile)
	mux.HandleFunc("/api/files/home", handleHomeDir)
	mux.HandleFunc("/api/server/files", handleListServerFiles)
	mux.HandleFunc("/api/server/files/content", handleServerFileContent)
}

func handleCurrentDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := r.URL.Query().Get("project_dir")
	if projectDir == "" {
		respondErr(w, http.StatusBadRequest, "project_dir is required")
		return
	}

	diffs, err := GetCurrentDiff(projectDir)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, diffs)
}

func handleSingleFileDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	diff, err := GetSingleFileDiff(projectDir, filePath)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, diff)
}

func handleCheckpoints(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		respondErr(w, http.StatusBadRequest, "project is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, err := ListCheckpoints(project)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var req CreateCheckpointRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.ProjectDir == "" {
			respondErr(w, http.StatusBadRequest, "project_dir is required")
			return
		}
		if len(req.FilePaths) == 0 {
			respondErr(w, http.StatusBadRequest, "file_paths is required")
			return
		}
		summary, err := CreateCheckpoint(project, req)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, summary)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCheckpointByID(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/checkpoints/{id} or /api/checkpoints/current
	path := strings.TrimPrefix(r.URL.Path, "/api/checkpoints/")

	project := r.URL.Query().Get("project")
	if project == "" {
		respondErr(w, http.StatusBadRequest, "project is required")
		return
	}

	// Handle /api/checkpoints/current
	if path == "current" {
		projectDir := r.URL.Query().Get("project_dir")
		if projectDir == "" {
			respondErr(w, http.StatusBadRequest, "project_dir is required")
			return
		}
		changes, err := GetCurrentChanges(project, projectDir)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, changes)
		return
	}

	// Handle /api/checkpoints/current/diff
	if path == "current/diff" {
		projectDir := r.URL.Query().Get("project_dir")
		if projectDir == "" {
			respondErr(w, http.StatusBadRequest, "project_dir is required")
			return
		}
		diffs, err := GetCurrentDiff(projectDir)
		if err != nil {
			respondErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, diffs)
		return
	}

	// Parse checkpoint ID, handling paths like "1" or "1/diff"
	idStr := path
	suffix := ""
	if slashIdx := strings.IndexByte(path, '/'); slashIdx >= 0 {
		idStr = path[:slashIdx]
		suffix = path[slashIdx+1:]
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondErr(w, http.StatusBadRequest, "invalid checkpoint id")
		return
	}

	// Handle /api/checkpoints/{id}/diff
	if suffix == "diff" {
		diffs, diffErr := GetCheckpointDiff(project, id)
		if diffErr != nil {
			respondErr(w, http.StatusNotFound, diffErr.Error())
			return
		}
		respondJSON(w, http.StatusOK, diffs)
		return
	}

	switch r.Method {
	case http.MethodGet:
		cp, err := GetCheckpoint(project, id)
		if err != nil {
			respondErr(w, http.StatusNotFound, err.Error())
			return
		}
		// Return without full content for listing
		type FileInfo struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		}
		type DetailResp struct {
			ID        int        `json:"id"`
			Name      string     `json:"name"`
			Timestamp string     `json:"timestamp"`
			Files     []FileInfo `json:"files"`
		}
		resp := DetailResp{ID: cp.ID, Name: cp.Name, Timestamp: cp.Timestamp}
		for _, f := range cp.Files {
			resp.Files = append(resp.Files, FileInfo{Path: f.Path, Status: f.Status})
		}
		respondJSON(w, http.StatusOK, resp)

	case http.MethodDelete:
		if err := DeleteCheckpoint(project, id); err != nil {
			respondErr(w, http.StatusNotFound, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func respondErr(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
