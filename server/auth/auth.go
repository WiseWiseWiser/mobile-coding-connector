package auth

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

const (
	cookieName      = "ai-critic-token"
	credentialsFile = ".server-credentials"
)

// loadCredentials reads the credentials file and returns all non-empty lines
func loadCredentials() (map[string]bool, error) {
	f, err := os.Open(credentialsFile)
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

// isValidToken checks if the given token matches any line in the credentials file.
// The file is re-read on each call so changes take effect immediately.
func isValidToken(token string) bool {
	if token == "" {
		return false
	}
	tokens, err := loadCredentials()
	if err != nil {
		return false
	}
	return tokens[token]
}

// Middleware returns an http.Handler that checks for a valid auth cookie.
// If the credentials file does not exist, all requests are allowed through (no auth required).
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

		// If credentials file doesn't exist, allow all (no auth configured)
		if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(cookieName)
		if err != nil || !isValidToken(cookie.Value) {
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

// RegisterAPI registers the login and auth check endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/auth/check", handleAuthCheck)
}

func handleAuthCheck(w http.ResponseWriter, _ *http.Request) {
	// If this handler is reached, the request has already passed the auth middleware
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
	if !isValidToken(req.Password) {
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
