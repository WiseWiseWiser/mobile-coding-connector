package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/ai"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/github"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

// initialDir stores the initial directory set via --dir flag
var initialDir string

// aiConfig stores the AI configuration
var aiConfig *config.Config

// SetInitialDir sets the initial directory for code review
func SetInitialDir(dir string) {
	initialDir = dir
}

// GetInitialDir returns the initial directory
func GetInitialDir() string {
	return initialDir
}

// SetAIConfig sets the AI configuration
func SetAIConfig(cfg *config.Config) {
	aiConfig = cfg
}

// GetAIConfig returns the AI configuration
func GetAIConfig() *config.Config {
	return aiConfig
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
	mux.HandleFunc("/api/review/commit", handleGitCommit)
	mux.HandleFunc("/api/review/push", handleGitPush)
	mux.HandleFunc("/api/review/fetch", handleGitFetch)
	mux.HandleFunc("/api/review/status", handleGitStatus)
	mux.HandleFunc("/api/review/branches", handleGitBranches)
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

	// Add providers and models from config
	if aiConfig != nil {
		for _, p := range aiConfig.GetAvailableProviders() {
			cfg.Providers = append(cfg.Providers, ProviderInfo{Name: p.Name})
		}
		for _, m := range aiConfig.GetAvailableModels() {
			cfg.Models = append(cfg.Models, ModelInfo{
				Provider:    m.Provider,
				Model:       m.Model,
				DisplayName: m.DisplayName,
			})
		}
		cfg.DefaultProvider = aiConfig.AI.DefaultProvider
		cfg.DefaultModel = aiConfig.AI.DefaultModel
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
	cmd := exec.Command("git", "add", req.Path)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
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

	cmd := exec.Command("git", "reset", "HEAD", req.Path)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to unstage file: %s", string(output))})
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
		cmd := exec.Command("git", "config", "user.name", req.UserName)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to set git user.name: %s", string(output))})
			return
		}
	}
	if req.UserEmail != "" {
		cmd := exec.Command("git", "config", "user.email", req.UserEmail)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to set git user.email: %s", string(output))})
			return
		}
	}

	cmd := exec.Command("git", "commit", "-m", req.Message)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
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

	cmd := exec.Command("git", "push")
	cmd.Dir = dir

	// Use SSH key if provided
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

		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyFile.Path)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	}

	if wantStream {
		// Use SSE streaming
		sseWriter := sse.NewWriter(w)
		if sseWriter == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Streaming not supported"})
			return
		}

		sseWriter.SendLog("Starting git push...")
		err := sseWriter.StreamCmd(cmd)
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

	cmd := exec.Command("git", "pull", "--ff-only")
	cmd.Dir = dir

	// Use SSH key if provided
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

		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyFile.Path)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	}

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
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}

	// Get branch name
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	branchOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %v", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Get status with porcelain format
	cmd = exec.Command("git", "status", "--porcelain=v1")
	cmd.Dir = dir
	output, err := cmd.Output()
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

		// Staged change
		if indexStatus != ' ' && indexStatus != '?' {
			status := parseStatusChar(indexStatus)
			result.Files = append(result.Files, GitStatusFile{
				Path:     filePath,
				Status:   status,
				IsStaged: true,
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
			})
		}
	}

	return result, nil
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
	cmd := exec.Command("git", "for-each-ref",
		"--sort=-committerdate",
		"--format=%(refname:short)\t%(committerdate:iso8601)\t%(HEAD)",
		"refs/heads/",
	)
	cmd.Dir = dir
	output, err := cmd.Output()
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
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}

	result := &GitDiffResult{
		Files: []DiffFile{},
	}

	// Get unstaged changes (working tree diff)
	cmd = exec.Command("git", "diff")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get working tree diff: %v", err)
	}
	result.WorkingTreeDiff = string(output)

	// Parse unstaged files
	unstagedFiles := parseGitDiff(string(output), false)
	result.Files = append(result.Files, unstagedFiles...)

	// Get staged changes
	cmd = exec.Command("git", "diff", "--cached")
	cmd.Dir = dir
	output, err = cmd.Output()
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
	if aiConfig != nil && req.Provider != "" && req.Model != "" {
		provider := aiConfig.GetProvider(req.Provider)
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
	} else if aiConfig != nil {
		baseURL, apiKey, model := aiConfig.GetDefaultAIConfig()
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
