package github

import (
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

	gossh "golang.org/x/crypto/ssh"

	"github.com/xhd2015/lifelog-private/ai-critic/server/encrypt"
	"github.com/xhd2015/lifelog-private/ai-critic/server/projects"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
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

	// Prepare SSH key if using SSH
	var keyFile *SSHKeyFile
	if req.UseSSH && req.SSHKey != "" {
		var err error
		keyFile, err = PrepareSSHKeyFile(req.SSHKey)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, CloneResponse{Status: "error", Error: err.Error()})
			return
		}
		defer keyFile.Cleanup()

		// Convert HTTPS URLs to SSH URLs when using SSH key
		// e.g. https://github.com/user/repo.git -> git@github.com:user/repo.git
		req.RepoURL = convertToSSHURL(req.RepoURL)
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

	if keyFile != nil {
		// Use GIT_SSH_COMMAND to specify the key; disable terminal prompt to prevent HTTPS fallback
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyFile.Path)
		cmd = exec.Command("git", "clone", "--progress", req.RepoURL, targetDir)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd), "GIT_TERMINAL_PROMPT=0")
	} else {
		cmd = exec.Command("git", "clone", "--progress", req.RepoURL, targetDir)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	}

	// Stream clone output via SSE
	sw := sse.NewWriter(w)
	if sw == nil {
		writeJSON(w, http.StatusInternalServerError, CloneResponse{Status: "error", Error: "Streaming not supported"})
		return
	}

	// Log the clone command for diagnostics
	sw.SendLog(fmt.Sprintf("$ git clone --progress %s %s", req.RepoURL, targetDir))
	if keyFile != nil {
		sw.SendLog(fmt.Sprintf("Using SSH key: %s (%d bytes)", keyFile.KeyType, keyFile.Size))
	}

	cloneErr := sw.StreamCmd(cmd)

	if cloneErr != nil {
		sw.SendError(fmt.Sprintf("Clone failed: %v", cloneErr))
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

	sw.SendDone(map[string]string{"dir": targetDir})
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

	// Set up SSE streaming
	sw := sse.NewWriter(w)
	if sw == nil {
		writeJSON(w, http.StatusInternalServerError, SSHTestResponse{Output: "Streaming not supported"})
		return
	}

	sw.SendLog(fmt.Sprintf("Testing SSH connection to git@%s...", req.Host))

	// Prepare SSH key (decrypt, normalize, validate, write to temp file)
	keyFile, err := PrepareSSHKeyFile(req.PrivateKey)
	if err != nil {
		sw.SendError(err.Error())
		sw.SendDone(map[string]string{"message": "SSH key preparation failed", "success": "false"})
		return
	}
	defer keyFile.Cleanup()

	sw.SendLog(fmt.Sprintf("SSH key validated: %s (%d bytes)", keyFile.KeyType, keyFile.Size))

	// Check if ssh is installed
	if _, lookErr := exec.LookPath("ssh"); lookErr != nil {
		sw.SendError("ssh is not installed. Please install openssh-client first (e.g. apt-get install -y openssh-client).")
		sw.SendDone(map[string]string{"message": "SSH connection failed: ssh not installed", "success": "false"})
		return
	}

	// Run ssh -vT git@<host> to test connection (verbose for diagnostics)
	cmd := exec.Command("ssh", "-v", "-T",
		"-i", keyFile.Path,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("git@%s", req.Host),
	)

	// Track success by inspecting output lines
	success := false
	err = sw.StreamCmdFunc(cmd, func(line string) bool {
		if strings.Contains(line, "successfully authenticated") ||
			strings.Contains(line, "Welcome to") ||
			strings.Contains(line, "Hi ") {
			success = true
		}
		return true // always send as log
	})

	// ssh -T to GitHub returns exit code 1 even on success
	if err != nil && !success {
		sw.SendDone(map[string]string{"message": "SSH connection failed", "success": "false"})
		return
	}

	sw.SendDone(map[string]string{"message": "SSH connection successful", "success": "true"})
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

// SSHKeyFile holds information about a prepared SSH key temp file.
type SSHKeyFile struct {
	Path    string // path to the temp file
	KeyType string // e.g. "ssh-rsa", "ssh-ed25519"
	Size    int    // key content size in bytes
}

// Cleanup removes the temp key file.
func (f *SSHKeyFile) Cleanup() {
	if f != nil && f.Path != "" {
		os.Remove(f.Path)
	}
}

// PrepareSSHKeyFile decrypts, normalizes, validates, and writes an SSH key to a temp file.
// Returns the prepared file info, or an error if any step fails.
func PrepareSSHKeyFile(encryptedKey string) (*SSHKeyFile, error) {
	// Decrypt
	plainKey, err := decryptSSHKey(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt SSH key: %v", err)
	}

	// Normalize line endings and ensure trailing newline
	keyContent := strings.ReplaceAll(plainKey, "\r\n", "\n")
	keyContent = strings.ReplaceAll(keyContent, "\r", "\n")
	keyContent = strings.TrimRight(keyContent, " \t\n") + "\n"

	// Validate key format using Go's SSH library
	signer, parseErr := gossh.ParsePrivateKey([]byte(keyContent))
	if parseErr != nil {
		return nil, fmt.Errorf("invalid SSH key: %v", parseErr)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "ai-critic-ssh-key-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.WriteString(keyContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to write SSH key: %v", err)
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0600)

	return &SSHKeyFile{
		Path:    tmpFile.Name(),
		KeyType: signer.PublicKey().Type(),
		Size:    len(keyContent),
	}, nil
}

// convertToSSHURL converts an HTTPS git URL to its SSH equivalent.
// e.g. https://github.com/user/repo.git -> git@github.com:user/repo.git
// Non-HTTPS URLs are returned unchanged.
func convertToSSHURL(repoURL string) string {
	parsed, err := url.Parse(repoURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return repoURL
	}

	host := parsed.Hostname()
	path := strings.TrimPrefix(parsed.Path, "/")
	if path == "" {
		return repoURL
	}

	return fmt.Sprintf("git@%s:%s", host, path)
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
