package github

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/encrypt"
	"github.com/xhd2015/lifelog-private/ai-critic/server/projects"
)

// OAuthConfig holds the GitHub OAuth configuration
type OAuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// OAuthTokenRequest is sent by the frontend to exchange a code for a token
type OAuthTokenRequest struct {
	Code string `json:"code"`
}

// OAuthTokenResponse is returned to the frontend
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
}

// RepoInfo represents a GitHub repository
type RepoInfo struct {
	FullName    string `json:"full_name"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Language    string `json:"language"`
	UpdatedAt   string `json:"updated_at"`
}

// CloneRequest is sent by the frontend to clone a repo
type CloneRequest struct {
	RepoURL   string `json:"repo_url"`
	TargetDir string `json:"target_dir"`
	// SSHKey is the private key content to use for SSH-based cloning
	SSHKey string `json:"ssh_key,omitempty"`
	// SSHKeyID is the ID of the SSH key used (for tracking)
	SSHKeyID string `json:"ssh_key_id,omitempty"`
	// UseSSH indicates whether to use SSH for cloning
	UseSSH bool `json:"use_ssh"`
}

// CloneResponse is returned after a clone operation
type CloneResponse struct {
	Status string `json:"status"`
	Dir    string `json:"dir,omitempty"`
	Error  string `json:"error,omitempty"`
}

var (
	oauthConfig   *OAuthConfig
	oauthConfigMu sync.RWMutex

	// configFilePath is where we persist the OAuth config
	configFilePath string
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configFilePath = filepath.Join(homeDir, ".ai-critic-github-oauth.json")
}

// loadConfigFromDisk loads saved OAuth config
func loadConfigFromDisk() {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return
	}
	var cfg OAuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	oauthConfig = &cfg
}

// saveConfigToDisk persists OAuth config
func saveConfigToDisk(cfg *OAuthConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFilePath, data, 0600)
}

// RegisterAPI registers the GitHub-related API endpoints
func RegisterAPI(mux *http.ServeMux) {
	// Load any saved config on startup
	loadConfigFromDisk()

	mux.HandleFunc("/api/github/oauth-config", handleOAuthConfig)
	mux.HandleFunc("/api/github/oauth-token", handleOAuthToken)
	mux.HandleFunc("/api/github/repos", handleListRepos)
	mux.HandleFunc("/api/github/clone", handleClone)
	mux.HandleFunc("/api/ssh-keys/test", handleTestSSHKey)
}

// handleOAuthConfig handles GET (retrieve) and POST (save) OAuth config
func handleOAuthConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetOAuthConfig(w, r)
	case http.MethodPost:
		handleSaveOAuthConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetOAuthConfig(w http.ResponseWriter, _ *http.Request) {
	oauthConfigMu.RLock()
	defer oauthConfigMu.RUnlock()

	resp := struct {
		Configured bool   `json:"configured"`
		ClientID   string `json:"client_id,omitempty"`
	}{}

	if oauthConfig != nil && oauthConfig.ClientID != "" {
		resp.Configured = true
		resp.ClientID = oauthConfig.ClientID
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleSaveOAuthConfig(w http.ResponseWriter, r *http.Request) {
	var cfg OAuthConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id and client_secret are required"})
		return
	}

	oauthConfigMu.Lock()
	oauthConfig = &cfg
	oauthConfigMu.Unlock()

	if err := saveConfigToDisk(&cfg); err != nil {
		fmt.Printf("[GitHub] Warning: failed to save OAuth config to disk: %v\n", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleOAuthToken exchanges an authorization code for an access token
func handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	oauthConfigMu.RLock()
	cfg := oauthConfig
	oauthConfigMu.RUnlock()

	if cfg == nil || cfg.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, OAuthTokenResponse{Error: "GitHub OAuth not configured"})
		return
	}

	var req OAuthTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, OAuthTokenResponse{Error: "Invalid request body"})
		return
	}

	if req.Code == "" {
		writeJSON(w, http.StatusBadRequest, OAuthTokenResponse{Error: "code is required"})
		return
	}

	// Exchange code for token with GitHub
	resp, err := http.PostForm("https://github.com/login/oauth/access_token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {req.Code},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, OAuthTokenResponse{Error: fmt.Sprintf("Failed to exchange code: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, OAuthTokenResponse{Error: "Failed to read GitHub response"})
		return
	}

	// Parse the response (GitHub returns form-encoded by default)
	values, err := url.ParseQuery(string(body))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, OAuthTokenResponse{Error: "Failed to parse GitHub response"})
		return
	}

	if errMsg := values.Get("error"); errMsg != "" {
		writeJSON(w, http.StatusBadRequest, OAuthTokenResponse{Error: fmt.Sprintf("%s: %s", errMsg, values.Get("error_description"))})
		return
	}

	tokenResp := OAuthTokenResponse{
		AccessToken: values.Get("access_token"),
		TokenType:   values.Get("token_type"),
		Scope:       values.Get("scope"),
	}

	writeJSON(w, http.StatusOK, tokenResp)
}

// handleListRepos lists the authenticated user's repos
func handleListRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.Header.Get("Authorization")
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authorization header required"})
		return
	}

	// Fetch repos from GitHub API
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/user/repos?per_page=100&sort=updated&affiliation=owner", nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to fetch repos: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeJSON(w, resp.StatusCode, map[string]string{"error": fmt.Sprintf("GitHub API error: %s", string(body))})
		return
	}

	var ghRepos []struct {
		FullName    string `json:"full_name"`
		CloneURL    string `json:"clone_url"`
		SSHURL      string `json:"ssh_url"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
		Language    string `json:"language"`
		UpdatedAt   string `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghRepos); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to parse repos"})
		return
	}

	repos := make([]RepoInfo, 0, len(ghRepos))
	for _, r := range ghRepos {
		repos = append(repos, RepoInfo{
			FullName:    r.FullName,
			CloneURL:    r.CloneURL,
			SSHURL:      r.SSHURL,
			Description: r.Description,
			Private:     r.Private,
			Language:    r.Language,
			UpdatedAt:   r.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, repos)
}

// handleClone clones a repository to a local directory
func handleClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CloneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, CloneResponse{Status: "error", Error: "Invalid request body"})
		return
	}

	if req.RepoURL == "" {
		writeJSON(w, http.StatusBadRequest, CloneResponse{Status: "error", Error: "repo_url is required"})
		return
	}

	// Decrypt the SSH key if it was encrypted
	if req.UseSSH && req.SSHKey != "" {
		decrypted, err := decryptSSHKey(req.SSHKey)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, CloneResponse{Status: "error", Error: fmt.Sprintf("Failed to decrypt SSH key: %v", err)})
			return
		}
		req.SSHKey = decrypted
	}

	targetDir := req.TargetDir
	if targetDir == "" {
		// Default to ./<repo-name> in current directory
		repoName := extractRepoName(req.RepoURL)
		targetDir = repoName
	}

	// Convert to absolute path for clarity
	if !filepath.IsAbs(targetDir) {
		absDir, err := filepath.Abs(targetDir)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, CloneResponse{Status: "error", Error: fmt.Sprintf("Failed to resolve path: %v", err)})
			return
		}
		targetDir = absDir
	}

	// Check if target directory already exists
	if _, err := os.Stat(targetDir); err == nil {
		writeJSON(w, http.StatusBadRequest, CloneResponse{Status: "error", Error: fmt.Sprintf("Directory already exists: %s", targetDir)})
		return
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, CloneResponse{Status: "error", Error: fmt.Sprintf("Failed to create directory: %v", err)})
		return
	}

	// Build git clone command
	var cmd *exec.Cmd
	var cleanupSSHKey func()

	if req.UseSSH && req.SSHKey != "" {
		// Write SSH key to a temp file
		tmpFile, err := os.CreateTemp("", "ai-critic-ssh-key-*")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, CloneResponse{Status: "error", Error: "Failed to create temp SSH key file"})
			return
		}
		if _, err := tmpFile.WriteString(req.SSHKey); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			writeJSON(w, http.StatusInternalServerError, CloneResponse{Status: "error", Error: "Failed to write SSH key"})
			return
		}
		tmpFile.Close()
		os.Chmod(tmpFile.Name(), 0600)

		cleanupSSHKey = func() {
			os.Remove(tmpFile.Name())
		}

		// Use GIT_SSH_COMMAND to specify the key
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", tmpFile.Name())
		cmd = exec.Command("git", "clone", "--progress", req.RepoURL, targetDir)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	} else {
		cmd = exec.Command("git", "clone", "--progress", req.RepoURL, targetDir)
	}

	// Stream clone output via SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, CloneResponse{Status: "error", Error: "Streaming not supported"})
		return
	}

	sendSSE := func(eventType string, data interface{}) {
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	// Combine stdout and stderr into a single pipe for streaming
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		if cleanupSSHKey != nil {
			cleanupSSHKey()
		}
		sendSSE("error", map[string]string{"type": "error", "message": fmt.Sprintf("Failed to start clone: %v", err)})
		return
	}

	// Wait for command in background, close pipe when done
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
		pw.Close()
	}()

	// Stream output lines - use custom split that handles \r (git progress uses \r)
	scanner := bufio.NewScanner(pr)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		// Find the earliest \n or \r
		for i, b := range data {
			if b == '\n' || b == '\r' {
				return i + 1, data[:i], nil
			}
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil // need more data
	})
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		sendSSE("log", map[string]string{"type": "log", "message": line})
	}
	pr.Close()

	// Get command exit status
	err := <-waitErr
	if cleanupSSHKey != nil {
		cleanupSSHKey()
	}

	if err != nil {
		sendSSE("error", map[string]string{"type": "error", "message": fmt.Sprintf("Clone failed: %v", err)})
		return
	}

	// Save project to store
	repoName := extractRepoName(req.RepoURL)
	if saveErr := projects.Add(projects.Project{
		Name:     repoName,
		RepoURL:  req.RepoURL,
		Dir:      targetDir,
		SSHKeyID: req.SSHKeyID,
		UseSSH:   req.UseSSH,
	}); saveErr != nil {
		fmt.Printf("[GitHub] Warning: failed to save project: %v\n", saveErr)
	}

	sendSSE("done", map[string]interface{}{"type": "done", "dir": targetDir})
}

// SSHTestRequest is sent by the frontend to test an SSH key
type SSHTestRequest struct {
	Host       string `json:"host"`
	PrivateKey string `json:"private_key"`
}

// SSHTestResponse is returned after testing an SSH key
type SSHTestResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
}

// handleTestSSHKey tests if an SSH key can connect to a target host
func handleTestSSHKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SSHTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SSHTestResponse{Output: "Invalid request body"})
		return
	}

	if req.Host == "" || req.PrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, SSHTestResponse{Output: "host and private_key are required"})
		return
	}

	// Decrypt the private key if it was encrypted
	privateKey, err := decryptSSHKey(req.PrivateKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, SSHTestResponse{Output: fmt.Sprintf("Failed to decrypt SSH key: %v", err)})
		return
	}
	req.PrivateKey = privateKey

	// Write key to temp file
	tmpFile, err := os.CreateTemp("", "ai-critic-ssh-test-*")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SSHTestResponse{Output: "Failed to create temp file"})
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(req.PrivateKey); err != nil {
		tmpFile.Close()
		writeJSON(w, http.StatusInternalServerError, SSHTestResponse{Output: "Failed to write SSH key"})
		return
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0600)

	// Run ssh -T git@<host> to test connection
	cmd := exec.Command("ssh", "-T",
		"-i", tmpFile.Name(),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("git@%s", req.Host),
	)

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	// ssh -T to GitHub returns exit code 1 even on success, with message like
	// "Hi user! You've successfully authenticated..."
	// So we check the output for success indicators
	success := false
	if err == nil {
		success = true
	} else if strings.Contains(outputStr, "successfully authenticated") ||
		strings.Contains(outputStr, "Welcome to") ||
		strings.Contains(outputStr, "Hi ") {
		success = true
	}

	writeJSON(w, http.StatusOK, SSHTestResponse{
		Success: success,
		Output:  outputStr,
	})
}

// decryptSSHKey attempts to decrypt an SSH key that was encrypted with the server's public key.
// If the key starts with "-----BEGIN", it's assumed to be unencrypted and returned as-is.
func decryptSSHKey(keyData string) (string, error) {
	// If it looks like a plain SSH key, return as-is
	if strings.HasPrefix(strings.TrimSpace(keyData), "-----BEGIN") {
		return keyData, nil
	}

	// Otherwise, try to decrypt
	decrypted, err := encrypt.Decrypt(keyData)
	if err != nil {
		return "", err
	}
	return decrypted, nil
}

func extractRepoName(repoURL string) string {
	// Handle SSH URLs like git@github.com:user/repo.git
	if strings.Contains(repoURL, ":") && strings.HasPrefix(repoURL, "git@") {
		parts := strings.SplitN(repoURL, ":", 2)
		if len(parts) == 2 {
			name := parts[1]
			name = strings.TrimSuffix(name, ".git")
			// Use the last part (repo name) only
			slashParts := strings.Split(name, "/")
			return slashParts[len(slashParts)-1]
		}
	}

	// Handle HTTPS URLs
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "repo"
	}
	name := filepath.Base(parsed.Path)
	name = strings.TrimSuffix(name, ".git")
	if name == "" || name == "." {
		return "repo"
	}
	return name
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
