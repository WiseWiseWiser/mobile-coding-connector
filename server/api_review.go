package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/ai"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/github"
	"github.com/xhd2015/lifelog-private/ai-critic/server/gitrunner"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

// initialDir stores the initial directory set via --dir flag
var initialDir string

// aiConfig stores the AI configuration (legacy)
var aiConfig *config.Config

// aiConfigAdapter stores the AI configuration (new)
var aiConfigAdapter *config.ConfigAdapter

// SetInitialDir sets the initial directory for code review
func SetInitialDir(dir string) {
	initialDir = dir
}

// GetInitialDir returns the initial directory
func GetInitialDir() string {
	return initialDir
}

// SetAIConfig sets the AI configuration (legacy, kept for backward compatibility)
func SetAIConfig(cfg *config.Config) {
	aiConfig = cfg
}

// GetAIConfig returns the AI configuration (legacy)
func GetAIConfig() *config.Config {
	return aiConfig
}

// SetAIConfigAdapter sets the AI configuration using the new adapter
func SetAIConfigAdapter(adapter *config.ConfigAdapter) {
	aiConfigAdapter = adapter
}

// GetAIConfigAdapter returns the AI configuration adapter
func GetAIConfigAdapter() *config.ConfigAdapter {
	return aiConfigAdapter
}

// getEffectiveAIConfig returns the effective AI config (adapter first, then legacy)
func getEffectiveAIConfig() *config.ConfigAdapter {
	if aiConfigAdapter != nil {
		return aiConfigAdapter
	}
	if aiConfig != nil {
		return config.NewConfigAdapter(&config.AIModelsConfig{
			Providers:       aiConfig.AI.Providers,
			Models:          aiConfig.AI.Models,
			DefaultProvider: aiConfig.AI.DefaultProvider,
			DefaultModel:    aiConfig.AI.DefaultModel,
		})
	}
	return nil
}

// CodeReviewRequest represents a request to review code changes
type CodeReviewRequest struct {
	Dir      string `json:"dir"`      // Directory to run git diff in, defaults to initial dir
	Provider string `json:"provider"` // AI provider to use (optional)
	Model    string `json:"model"`    // AI model to use (optional)
	SSHKey   string `json:"ssh_key"`  // Encrypted SSH private key for git operations (optional)
}

// GitDiffResult holds the result of git diff commands
type GitDiffResult struct {
	WorkingTreeDiff string     `json:"workingTreeDiff"` // Unstaged changes (raw diff)
	StagedDiff      string     `json:"stagedDiff"`      // Staged changes (raw diff)
	Files           []DiffFile `json:"files"`           // Parsed file diffs
}

// DiffFile represents a single file's diff
type DiffFile struct {
	Path       string `json:"path"`       // File path
	Status     string `json:"status"`     // "modified", "added", "deleted"
	OldPath    string `json:"oldPath"`    // For renamed files
	Diff       string `json:"diff"`       // The diff content for this file
	IsStaged   bool   `json:"isStaged"`   // Whether this is a staged change
	TotalLines int    `json:"totalLines"` // Total lines in the file
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"` // Message content
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`    // Chat history
	DiffContext string        `json:"diffContext"` // The diff context for the chat
	Provider    string        `json:"provider"`    // AI provider to use
	Model       string        `json:"model"`       // AI model to use
}

func registerReviewAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/review/config", handleGetConfig)
	mux.HandleFunc("/api/review/diff", handleGetDiff)
	mux.HandleFunc("/api/review/chat", handleChat)
	mux.HandleFunc("/api/review/stage", handleStageFile)
	mux.HandleFunc("/api/review/unstage", handleUnstageFile)
	mux.HandleFunc("/api/review/checkout", handleGitCheckout)
	mux.HandleFunc("/api/review/remove", handleGitRemove)
	mux.HandleFunc("/api/review/commit", handleGitCommit)
	mux.HandleFunc("/api/review/push", handleGitPush)
	mux.HandleFunc("/api/review/fetch", handleGitFetch)
	mux.HandleFunc("/api/review/status", handleGitStatus)
	mux.HandleFunc("/api/review/branches", handleGitBranches)
	mux.HandleFunc("/api/review/list-untracked-dir", handleListUntrackedDir)
	mux.HandleFunc("/api/review/generate-commit-message", handleGenerateCommitMessage)
}

// ProviderInfo represents a provider for the frontend
type ProviderInfo struct {
	Name string `json:"name"`
}

// ModelInfo represents a model for the frontend
type ModelInfo struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	DisplayName string `json:"displayName,omitempty"`
}

// ConfigInfo represents the configuration for the frontend
type ConfigInfo struct {
	InitialDir      string         `json:"initialDir"`
	Providers       []ProviderInfo `json:"providers"`
	Models          []ModelInfo    `json:"models"`
	DefaultProvider string         `json:"defaultProvider,omitempty"`
	DefaultModel    string         `json:"defaultModel,omitempty"`
}

// handleGetConfig returns the initial configuration including the default directory
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := ConfigInfo{
		InitialDir: initialDir,
	}

	// Add providers and models from config (use adapter if available)
	effectiveCfg := getEffectiveAIConfig()
	if effectiveCfg != nil {
		for _, p := range effectiveCfg.GetAvailableProviders() {
			cfg.Providers = append(cfg.Providers, ProviderInfo{Name: p.Name})
		}
		for _, m := range effectiveCfg.GetAvailableModels() {
			cfg.Models = append(cfg.Models, ModelInfo{
				Provider:    m.Provider,
				Model:       m.Model,
				DisplayName: m.DisplayName,
			})
		}
		cfg.DefaultProvider = effectiveCfg.GetDefaultProvider()
		cfg.DefaultModel = effectiveCfg.GetDefaultModel()
	}

	writeJSON(w, http.StatusOK, cfg)
}

// handleGetDiff returns the git diff for the specified directory
func handleGetDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CodeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := req.Dir
	if dir == "" {
		dir = initialDir
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get current directory"})
				return
			}
		}
	}

	result, err := getGitDiff(dir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// StageFileRequest represents a request to stage a file
type StageFileRequest struct {
	Dir  string `json:"dir"`  // Directory to run git add in
	Path string `json:"path"` // File path to stage
}

// handleStageFile handles requests to stage a file using git add
func handleStageFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req StageFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := req.Dir
	if dir == "" {
		dir = initialDir
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get current directory"})
				return
			}
		}
	}

	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "File path is required"})
		return
	}

	// Run git add
	output, err := gitrunner.Add(req.Path).Dir(dir).Run()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to stage file: %s", string(output))})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleUnstageFile handles requests to unstage a file using git reset HEAD
func handleUnstageFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req StageFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "File path is required"})
		return
	}

	output, err := gitrunner.Reset(req.Path).Dir(dir).Run()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to unstage file: %s", string(output))})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleGitCheckout handles requests to discard changes in working tree using git checkout --
func handleGitCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req StageFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "File path is required"})
		return
	}

	output, err := gitrunner.Checkout(req.Path).Dir(dir).Run()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to checkout file: %s", string(output))})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RemoveFileRequest represents a request to remove a file
type RemoveFileRequest struct {
	Dir  string `json:"dir"`  // Directory to run rm in
	Path string `json:"path"` // File path to remove
}

// handleGitRemove handles requests to remove an untracked file using rm -f
func handleGitRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req RemoveFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "File path is required"})
		return
	}

	filePath := filepath.Join(dir, req.Path)
	if err := os.Remove(filePath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to remove file: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GitCommitRequest represents a request to commit changes
type GitCommitRequest struct {
	Dir       string `json:"dir"`
	Message   string `json:"message"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// handleGitCommit handles requests to commit staged changes
func handleGitCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req GitCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Commit message is required"})
		return
	}

	// Set git user config if provided
	if req.UserName != "" {
		if output, err := gitrunner.Config("user.name", req.UserName).Dir(dir).Run(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to set git user.name: %s", string(output))})
			return
		}
	}
	if req.UserEmail != "" {
		if output, err := gitrunner.Config("user.email", req.UserEmail).Dir(dir).Run(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to set git user.email: %s", string(output))})
			return
		}
	}

	output, err := gitrunner.Commit(req.Message).Dir(dir).Run()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to commit: %s", string(output))})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "output": string(output)})
}

// handleGitPush handles requests to push to remote with SSE streaming
func handleGitPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req CodeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	// Check if client wants SSE streaming
	acceptHeader := r.Header.Get("Accept")
	wantStream := acceptHeader == "text/event-stream"

	// Get current branch first
	branch, err := gitrunner.GetCurrentBranch(dir)
	if err != nil {
		if wantStream {
			sseWriter := sse.NewWriter(w)
			if sseWriter != nil {
				sseWriter.SendError(fmt.Sprintf("Failed to get current branch: %v", err))
				sseWriter.SendDone(map[string]string{"success": "false"})
			}
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get current branch: %v", err)})
		return
	}

	// Build git push command using gitrunner
	var keyPath string
	if req.SSHKey != "" {
		keyFile, err := github.PrepareSSHKeyFile(req.SSHKey)
		if err != nil {
			if wantStream {
				sseWriter := sse.NewWriter(w)
				if sseWriter != nil {
					sseWriter.SendError(fmt.Sprintf("Failed to prepare SSH key: %v", err))
					sseWriter.SendDone(map[string]string{"success": "false"})
				}
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Failed to prepare SSH key: %v", err)})
			return
		}
		defer keyFile.Cleanup()
		keyPath = keyFile.Path
	}
	cmd := gitrunner.Push(branch, keyPath).Dir(dir).Exec()

	if wantStream {
		// Use SSE streaming
		sseWriter := sse.NewWriter(w)
		if sseWriter == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Streaming not supported"})
			return
		}

		sseWriter.SendLog(fmt.Sprintf("Starting git push origin HEAD:%s...", branch))
		err = sseWriter.StreamCmd(cmd)
		if err != nil {
			sseWriter.SendError(fmt.Sprintf("Push failed: %v", err))
			sseWriter.SendDone(map[string]string{"success": "false"})
			return
		}
		sseWriter.SendDone(map[string]string{"success": "true", "message": "Push completed successfully"})
		return
	}

	// Non-streaming fallback
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to push: %s", string(output))})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "output": string(output)})
}

func handleGitFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req CodeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	// Check if client wants SSE streaming
	acceptHeader := r.Header.Get("Accept")
	wantStream := acceptHeader == "text/event-stream"

	// Build git pull command using gitrunner
	var keyPath string
	if req.SSHKey != "" {
		keyFile, err := github.PrepareSSHKeyFile(req.SSHKey)
		if err != nil {
			if wantStream {
				sseWriter := sse.NewWriter(w)
				if sseWriter != nil {
					sseWriter.SendError(fmt.Sprintf("Failed to prepare SSH key: %v", err))
					sseWriter.SendDone(map[string]string{"success": "false"})
				}
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Failed to prepare SSH key: %v", err)})
			return
		}
		defer keyFile.Cleanup()
		keyPath = keyFile.Path
	}
	cmd := gitrunner.PullFFOnly(keyPath).Dir(dir).Exec()

	if wantStream {
		sseWriter := sse.NewWriter(w)
		if sseWriter == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Streaming not supported"})
			return
		}

		sseWriter.SendLog("Starting git pull --ff-only...")
		err := sseWriter.StreamCmd(cmd)
		if err != nil {
			sseWriter.SendError(fmt.Sprintf("Pull failed: %v", err))
			sseWriter.SendDone(map[string]string{"success": "false"})
			return
		}
		sseWriter.SendDone(map[string]string{"success": "true", "message": "Pull completed successfully"})
		return
	}

	// Non-streaming fallback
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to pull: %s", string(output))})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "output": string(output)})
}

// GitStatusFile represents a single file in git status output
type GitStatusFile struct {
	Path     string `json:"path"`
	Status   string `json:"status"`   // "added", "modified", "deleted", "renamed", "untracked"
	IsStaged bool   `json:"isStaged"` // Whether the change is staged
	Size     int64  `json:"size"`     // File size in bytes
	IsDir    bool   `json:"isDir"`    // Whether this is a directory
}

// GitStatusResult represents the result of git status
type GitStatusResult struct {
	Branch string          `json:"branch"`
	Files  []GitStatusFile `json:"files"`
}

// handleGitStatus returns the git status with separated staged/unstaged files
func handleGitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req CodeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	result, err := getGitStatus(dir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListUntrackedDirRequest represents a request to list contents of an untracked directory
type ListUntrackedDirRequest struct {
	Dir        string `json:"dir"`        // Git repository directory
	SubDirPath string `json:"subDirPath"` // Path within the untracked directory to list
}

// handleListUntrackedDir lists contents of an untracked directory for navigation
func handleListUntrackedDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req ListUntrackedDirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	fullPath := filepath.Join(dir, req.SubDirPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to read directory: %v", err)})
		return
	}

	var files []GitStatusFile
	for _, entry := range entries {
		entryPath := filepath.Join(req.SubDirPath, entry.Name())

		// Skip files/dirs that are ignored by git
		if gitrunner.IsIgnored(dir, entryPath) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, GitStatusFile{
			Path:     entryPath,
			Status:   "untracked",
			IsStaged: false,
			Size:     info.Size(),
			IsDir:    entry.IsDir(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"files": files})
}

// resolveDir resolves the git directory from the request, falling back to initialDir or cwd
func resolveDir(dir string) string {
	if dir != "" {
		return dir
	}
	if initialDir != "" {
		return initialDir
	}
	d, err := os.Getwd()
	if err != nil {
		return ""
	}
	return d
}

// getGitStatus runs git status --porcelain=v1 -b and parses the output
func getGitStatus(dir string) (*GitStatusResult, error) {
	// Check if directory is a git repository
	if err := gitrunner.RevParse("--git-dir").Dir(dir).RunSilent(); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}

	// Get branch name
	branchOutput, err := gitrunner.Branch("--show-current").Dir(dir).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %v", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Get status with porcelain format
	output, err := gitrunner.Status("--porcelain=v1").Dir(dir).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %v", err)
	}

	result := &GitStatusResult{
		Branch: branch,
		Files:  []GitStatusFile{},
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		indexStatus := line[0]    // staged status
		workTreeStatus := line[1] // unstaged status
		filePath := strings.TrimSpace(line[3:])

		// Handle renamed files - format is "old -> new"
		if idx := strings.Index(filePath, " -> "); idx >= 0 {
			filePath = filePath[idx+4:]
		}

		// Get file size and check if directory
		size, isDir := getFileSize(dir, filePath)

		// Staged change
		if indexStatus != ' ' && indexStatus != '?' {
			status := parseStatusChar(indexStatus)
			result.Files = append(result.Files, GitStatusFile{
				Path:     filePath,
				Status:   status,
				IsStaged: true,
				Size:     size,
				IsDir:    isDir,
			})
		}

		// Unstaged change
		if workTreeStatus != ' ' {
			status := parseStatusChar(workTreeStatus)
			if workTreeStatus == '?' {
				status = "untracked"
			}
			result.Files = append(result.Files, GitStatusFile{
				Path:     filePath,
				Status:   status,
				IsStaged: false,
				Size:     size,
				IsDir:    isDir,
			})
		}
	}

	return result, nil
}

// getFileSize returns the size of a file in bytes and whether it's a directory
func getFileSize(dir, filePath string) (int64, bool) {
	fullPath := filepath.Join(dir, filePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, false
	}
	return info.Size(), info.IsDir()
}

// parseStatusChar converts a git status character to a human-readable status
func parseStatusChar(c byte) string {
	switch c {
	case 'A':
		return "added"
	case 'M':
		return "modified"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case '?':
		return "untracked"
	default:
		return "modified"
	}
}

// GitBranch represents a git branch
type GitBranch struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"isCurrent"`
	Date      string `json:"date"` // ISO date of last commit
}

// handleGitBranches returns branches sorted by recent commit date
func handleGitBranches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req CodeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	branches, err := getGitBranches(dir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, branches)
}

// getGitBranches returns local branches sorted by most recent commit date
func getGitBranches(dir string) ([]GitBranch, error) {
	// Use git for-each-ref to list branches sorted by -committerdate (most recent first)
	output, err := gitrunner.ForEachRef(
		"--sort=-committerdate",
		"--format=%(refname:short)\t%(committerdate:iso8601)\t%(HEAD)",
		"refs/heads/",
	).Dir(dir).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %v", err)
	}

	var branches []GitBranch
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		branches = append(branches, GitBranch{
			Name:      parts[0],
			Date:      strings.TrimSpace(parts[1]),
			IsCurrent: strings.TrimSpace(parts[2]) == "*",
		})
	}

	return branches, nil
}

// getGitDiff runs git diff commands and returns the results
func getGitDiff(dir string) (*GitDiffResult, error) {
	// Check if directory is a git repository
	if err := gitrunner.RevParse("--git-dir").Dir(dir).RunSilent(); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}

	result := &GitDiffResult{
		Files: []DiffFile{},
	}

	// Get unstaged changes (working tree diff)
	output, err := gitrunner.Diff().Dir(dir).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get working tree diff: %v", err)
	}
	result.WorkingTreeDiff = string(output)

	// Parse unstaged files
	unstagedFiles := parseGitDiff(string(output), false)
	result.Files = append(result.Files, unstagedFiles...)

	// Get staged changes
	output, err = gitrunner.DiffCached().Dir(dir).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %v", err)
	}
	result.StagedDiff = string(output)

	// Parse staged files
	stagedFiles := parseGitDiff(string(output), true)
	result.Files = append(result.Files, stagedFiles...)

	// Count total lines for each file
	for i := range result.Files {
		file := &result.Files[i]
		if file.Status == "deleted" {
			file.TotalLines = 0
			continue
		}
		filePath := filepath.Join(dir, file.Path)
		lineCount, err := countFileLines(filePath)
		if err != nil {
			// If we can't count lines, just set to 0
			file.TotalLines = 0
		} else {
			file.TotalLines = lineCount
		}
	}

	return result, nil
}

// countFileLines counts the number of lines in a file
func countFileLines(filePath string) (int, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	if len(content) == 0 {
		return 0, nil
	}
	lines := bytes.Count(content, []byte("\n"))
	// If file doesn't end with newline, add 1 for the last line
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines, nil
}

// parseGitDiff parses a git diff output into individual file diffs
func parseGitDiff(diffOutput string, isStaged bool) []DiffFile {
	var files []DiffFile
	if diffOutput == "" {
		return files
	}

	// Split by "diff --git" to get individual file diffs
	parts := strings.Split(diffOutput, "diff --git ")
	for _, part := range parts {
		if part == "" {
			continue
		}

		// Re-add the "diff --git " prefix for the full diff
		fullDiff := "diff --git " + part

		// Parse the file path from the first line
		lines := strings.SplitN(part, "\n", 2)
		if len(lines) == 0 {
			continue
		}

		// Parse "a/path b/path" format
		firstLine := lines[0]
		pathParts := strings.Fields(firstLine)
		if len(pathParts) < 2 {
			continue
		}

		// Remove "a/" and "b/" prefixes
		aPath := strings.TrimPrefix(pathParts[0], "a/")
		bPath := strings.TrimPrefix(pathParts[1], "b/")

		// Determine status
		status := "modified"
		if strings.Contains(part, "new file mode") {
			status = "added"
		} else if strings.Contains(part, "deleted file mode") {
			status = "deleted"
		} else if aPath != bPath {
			status = "renamed"
		}

		files = append(files, DiffFile{
			Path:     bPath,
			OldPath:  aPath,
			Status:   status,
			Diff:     fullDiff,
			IsStaged: isStaged,
		})
	}

	return files
}

// rulesDir is the directory containing review rules
var rulesDir = "rules"

// SetRulesDir sets the directory for review rules
func SetRulesDir(dir string) {
	rulesDir = dir
}

// loadReviewRules reads the REVIEW_RULES.md file
func loadReviewRules() string {
	rulesFile := rulesDir + "/REVIEW_RULES.md"
	content, err := os.ReadFile(rulesFile)
	if err != nil {
		fmt.Printf("[Review] Warning: Could not read rules file %s: %v\n", rulesFile, err)
		return ""
	}
	return string(content)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// handleChat handles streaming chat requests
func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Log request for debugging
	fmt.Printf("[Chat] Request received: provider=%s, model=%s, messages=%d, diffContext=%d bytes\n",
		req.Provider, req.Model, len(req.Messages), len(req.DiffContext))

	// Get AI config
	var cfg ai.Config
	effectiveCfg := getEffectiveAIConfig()
	if effectiveCfg != nil && req.Provider != "" && req.Model != "" {
		provider := effectiveCfg.GetProvider(req.Provider)
		if provider == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Unknown provider: %s", req.Provider)})
			return
		}
		cfg = ai.Config{
			Provider: ai.ProviderOpenAI,
			APIKey:   provider.APIKey,
			BaseURL:  provider.BaseURL,
			Model:    req.Model,
		}
	} else if effectiveCfg != nil {
		baseURL, apiKey, model := effectiveCfg.GetDefaultAIConfig()
		cfg = ai.Config{
			Provider: ai.ProviderOpenAI,
			APIKey:   apiKey,
			BaseURL:  baseURL,
			Model:    model,
		}
	} else {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "API key not configured"})
			return
		}
		cfg = ai.Config{
			Provider: ai.ProviderOpenAI,
			APIKey:   apiKey,
			Model:    os.Getenv("OPENAI_MODEL"),
		}
		if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
			cfg.BaseURL = baseURL
		}
	}

	if cfg.APIKey == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "API key not configured"})
		return
	}

	// Build messages with system context
	rules := loadReviewRules()
	var systemPrompt string
	if rules != "" {
		systemPrompt = `You are a code review assistant. Code changes (git diff):

` + req.DiffContext + `

Review rules to check:

` + rules + `

STRICT RULES:
- ONLY report rule violations, nothing else
- NO "good practices observed", NO "additional observations", NO suggestions beyond the rules
- Be BRIEF: [file]: [rule violated] - [one-line fix]
- If no violations, just say "No issues found."`
	} else {
		systemPrompt = `You are a code review assistant. Code changes (git diff):

` + req.DiffContext + `

Be concise and helpful.`
	}

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
	}
	for _, msg := range req.Messages {
		messages = append(messages, ai.Message{Role: msg.Role, Content: msg.Content})
	}

	// Set up SSE streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Streaming not supported"})
		return
	}

	fmt.Printf("[Chat] Starting stream with model: %s, baseURL: %s\n", cfg.Model, cfg.BaseURL)

	// Stream the response
	err := ai.CallStream(r.Context(), cfg, messages, func(chunk ai.StreamChunk) error {
		if chunk.Content != "" {
			data, _ := json.Marshal(map[string]interface{}{
				"type":    string(chunk.Type),
				"content": chunk.Content,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		return nil
	})

	if err != nil {
		fmt.Printf("[Chat] Stream error: %v\n", err)
		data, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	fmt.Printf("[Chat] Stream completed\n")
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

type GenerateCommitMessageRequest struct {
	Dir string `json:"dir"`
}

type GenerateCommitMessageResponse struct {
	Message string `json:"message"`
}

func handleGenerateCommitMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateCommitMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	dir := resolveDir(req.Dir)
	if dir == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve directory"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Streaming not supported"})
		return
	}

	sendLog := func(msg string) {
		data, _ := json.Marshal(map[string]string{"type": "log", "message": msg})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	sendError := func(msg string) {
		data, _ := json.Marshal(map[string]string{"type": "error", "message": msg})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	sendDone := func(msg string) {
		data, _ := json.Marshal(map[string]string{"type": "done", "message": msg})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	sendLog("$ git diff --cached")
	stagedDiffOutput, err := gitrunner.DiffCached().Dir(dir).Output()
	if err != nil {
		sendError(fmt.Sprintf("Failed to get staged diff: %v", err))
		sendDone("")
		return
	}

	stagedDiff := string(stagedDiffOutput)
	if stagedDiff == "" {
		sendError("No staged changes to generate commit message for")
		sendDone("")
		return
	}

	fileCount := strings.Count(stagedDiff, "diff --git")
	if fileCount == 0 && len(stagedDiff) > 0 {
		fileCount = 1
	}

	sendLog(fmt.Sprintf("Staged files: %d, Diff length: %d chars", fileCount, len(stagedDiff)))
	sendLog(fmt.Sprintf("Passing diff to agent..."))

	commitPrompt := fmt.Sprintf(`Generate a brief git commit message (1 line title, max 50 characters, plus a short description if needed) for the following staged changes (git diff). Focus on what changed and why.

Git diff:
%s

Respond with ONLY the commit message in this format:
Title: <short title>
Description: <optional short description>`, stagedDiff)

	sendLog("$ opencode models")
	freeModels, selectedModel, err := findFreeModel()
	if err != nil {
		sendLog(fmt.Sprintf("Warning: Could not get free models: %v", err))
	} else {
		sendLog(fmt.Sprintf("Free models: %s", strings.Join(freeModels, ", ")))
		if selectedModel != "" {
			sendLog(fmt.Sprintf("Using model: %s", selectedModel))
		}
	}

	var args []string
	promptSummary := fmt.Sprintf("\"Generate brief git commit message (title + optional desc) for %d staged file(s), %d chars\"", fileCount, len(stagedDiff))
	if selectedModel != "" {
		args = []string{"run", commitPrompt, "--model", selectedModel, "--format", "json"}
		sendLog(fmt.Sprintf("$ opencode run %s --model %s --format json", promptSummary, selectedModel))
	} else {
		args = []string{"run", commitPrompt, "--format", "json"}
		sendLog(fmt.Sprintf("$ opencode run %s --format json", promptSummary))
	}

	sendLog("Running agent...")

	cmd, err := tool_exec.New("opencode", args, &tool_exec.Options{
		Dir: dir,
	})
	if err != nil {
		sendError(fmt.Sprintf("Failed to run opencode: %v", err))
		sendDone("")
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendError(fmt.Sprintf("Failed to create stdout pipe: %v", err))
		sendDone("")
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sendError(fmt.Sprintf("Failed to create stderr pipe: %v", err))
		sendDone("")
		return
	}

	if err := cmd.Start(); err != nil {
		sendError(fmt.Sprintf("Failed to start opencode: %v", err))
		sendDone("")
		return
	}

	var fullOutput strings.Builder
	doneChan := make(chan struct{})

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				line := string(buf[:n])
				sendLog(line)
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				line := string(buf[:n])
				fullOutput.WriteString(line)
				sendLog(line)
			}
			if err != nil {
				break
			}
		}
		doneChan <- struct{}{}
	}()

	<-doneChan

	err = cmd.Wait()
	output := fullOutput.String()

	if err != nil {
		sendError(fmt.Sprintf("Failed to generate commit message: %v", err))
		sendDone("")
		return
	}

	commitMessage := parseOpencodeJSONOutput(output)
	if commitMessage == "" {
		sendError("Failed to parse commit message from opencode output")
		sendDone("")
		return
	}

	commitMessage = strings.TrimPrefix(commitMessage, "Title:")
	commitMessage = strings.TrimPrefix(commitMessage, "title:")
	commitMessage = strings.TrimSpace(commitMessage)

	sendDone(commitMessage)
}

func parseOpencodeJSONOutput(output string) string {
	lines := strings.Split(output, "\n")
	var fullText strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		eventType, ok := event["type"].(string)
		if !ok || eventType != "text" {
			continue
		}
		part, ok := event["part"].(map[string]interface{})
		if !ok {
			continue
		}
		text, ok := part["text"].(string)
		if !ok {
			continue
		}
		fullText.WriteString(text)
	}
	return strings.TrimSpace(fullText.String())
}

func findFreeModel() (freeModels []string, selectedModel string, err error) {
	cmd, err := tool_exec.New("opencode", []string{"models"}, nil)
	if err != nil {
		return nil, "", err
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", err
	}

	models := strings.Split(string(output), "\n")
	for _, model := range models {
		model = strings.TrimSpace(model)
		if strings.Contains(model, "free") || strings.HasPrefix(model, "opencode/") && strings.Contains(model, "-free") {
			freeModels = append(freeModels, model)
		}
	}

	if len(freeModels) > 0 {
		selectedModel = freeModels[0]
	}
	return freeModels, selectedModel, nil
}
