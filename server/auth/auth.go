package auth

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

const cookieName = "ai-critic-token"

var (
	credentialsFileMu   sync.RWMutex
	credentialsFilePath = config.CredentialsFile
)

// SetCredentialsFile sets the path to the credentials file.
// Must be called before the server starts.
func SetCredentialsFile(path string) {
	credentialsFileMu.Lock()
	defer credentialsFileMu.Unlock()
	credentialsFilePath = path
}

func getCredentialsFile() string {
	credentialsFileMu.RLock()
	defer credentialsFileMu.RUnlock()
	return credentialsFilePath
}

// loadCredentials reads the credentials file and returns all non-empty lines
func loadCredentials() (map[string]bool, error) {
	f, err := os.Open(getCredentialsFile())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tokens := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		tokens[line] = true
	}
	return tokens, scanner.Err()
}

// loadAndCheckToken loads the credentials file once and returns whether
// the server is initialized and whether the given token is valid.
func loadAndCheckToken(token string) (initialized bool, valid bool) {
	tokens, err := loadCredentials()
	if err != nil || len(tokens) == 0 {
		return false, false
	}
	if token == "" {
		return true, false
	}
	return true, tokens[token]
}

// Middleware returns an http.Handler that checks for a valid auth cookie.
// When the server is not initialized (credentials file missing or empty),
// API requests return a "not_initialized" error so the frontend can show setup UI.
// Requests to skipPaths are always allowed through without auth.
func Middleware(next http.Handler, skipPaths []string) http.Handler {
	skipSet := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		skipSet[p] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check auth for /api/* paths
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for allowed paths
		if skipSet[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// Load credentials once: check initialization and token validity
		var token string
		if cookie, err := r.Cookie(cookieName); err == nil {
			token = cookie.Value
		}
		initialized, valid := loadAndCheckToken(token)

		if !initialized {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "not_initialized"})
			return
		}

		if !valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SetupRequest represents the initial credential setup request body
type SetupRequest struct {
	Credential string `json:"credential"`
}

// MaskedCredential is a credential entry with its value masked for display.
type MaskedCredential struct {
	Masked string `json:"masked"` // e.g. "1a******ed"
}

// maskToken masks a token, showing first 2 and last 2 characters.
func maskToken(token string) string {
	if len(token) <= 4 {
		return strings.Repeat("*", len(token))
	}
	return token[:2] + strings.Repeat("*", len(token)-4) + token[len(token)-2:]
}

// ListMaskedCredentials returns all credential tokens with masked values.
func ListMaskedCredentials() ([]MaskedCredential, error) {
	tokens, err := loadCredentials()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []MaskedCredential
	for t := range tokens {
		result = append(result, MaskedCredential{Masked: maskToken(t)})
	}
	return result, nil
}

// ExportCredentials returns all raw credential tokens for export.
func ExportCredentials() ([]string, error) {
	tokens, err := loadCredentials()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []string
	for t := range tokens {
		result = append(result, t)
	}
	return result, nil
}

// ImportCredentials merges new tokens into the credentials file, deduplicating.
func ImportCredentials(newTokens []string) error {
	existing, err := loadCredentials()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if existing == nil {
		existing = make(map[string]bool)
	}

	for _, t := range newTokens {
		t = strings.TrimSpace(t)
		if t != "" {
			existing[t] = true
		}
	}

	// Rebuild the file
	var lines []string
	for t := range existing {
		lines = append(lines, t)
	}

	credFile := getCredentialsFile()
	if err := os.MkdirAll(filepath.Dir(credFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(credFile, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

// RegisterAPI registers the login and auth check endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/auth/check", handleAuthCheck)
	mux.HandleFunc("/api/auth/setup", handleSetup)
	mux.HandleFunc("/api/auth/credentials", handleListCredentials)
	mux.HandleFunc("/api/auth/credentials/add", handleAddCredential)
	mux.HandleFunc("/api/auth/credentials/generate", handleGenerateCredential)
}

func handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check initialization and token validity in one read
	var token string
	if cookie, err := r.Cookie(cookieName); err == nil {
		token = cookie.Value
	}
	initialized, valid := loadAndCheckToken(token)

	if !initialized {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not_initialized"})
		return
	}
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only allow setup when server is not initialized
	initialized, _ := loadAndCheckToken("")
	if initialized {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "server already initialized"})
		return
	}

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Credential == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "credential is required"})
		return
	}

	// Write the credential to the credentials file
	credFile := getCredentialsFile()
	if err := os.MkdirAll(filepath.Dir(credFile), 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create data directory"})
		return
	}
	if err := os.WriteFile(credFile, []byte(req.Credential+"\n"), 0600); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to write credentials file"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleListCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	creds, err := ListMaskedCredentials()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if creds == nil {
		creds = []MaskedCredential{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"credentials": creds})
}

func handleAddCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "token is required"})
		return
	}

	if err := ImportCredentials([]string{token}); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGenerateCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate 32 random bytes, then SHA-256 hash to produce a 64-char hex credential
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to generate random bytes: %v", err)})
		return
	}
	hash := sha256.Sum256(raw)
	credential := hex.EncodeToString(hash[:])

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"credential": credential})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "username and password are required"})
		return
	}

	// Password must match any line in the credentials file
	_, valid := loadAndCheckToken(req.Password)
	if !valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
		return
	}

	// Set cookie with the password as the token
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    req.Password,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   365 * 24 * 3600, // 1 year
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
