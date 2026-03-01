package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/less-gen/flags"
)

const help = `Usage: basic_auth_proxy [options]

A proxy that adds Basic Auth headers to backend requests.
Uses cookie-based authentication with encrypted tokens.

Options:
  --port PORT          Port to listen on (required)
  --backend-port PORT  Port to proxy to (required)
  -h, --help           Show this help message

The proxy validates credentials by testing against the backend.
If the backend returns 401, login fails. Otherwise, a session
token is created and stored in an encrypted cookie.

Token expiration: 7 days (auto-extended on activity)
`

const cookieName = "basic-auth-token"
const tokenDuration = 7 * 24 * time.Hour

var configDir = ".ai-critic"
var configFile = "basic-auth-config.json"

type tokenData struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CreatedAt int64  `json:"created_at"`
}

type config struct {
	SecretKey string `json:"secret_key"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var port int
	var backendPort int

	args, err := flags.
		Int("--port", &port).
		Int("--backend-port", &backendPort).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return err
	}

	if port == 0 {
		return fmt.Errorf("--port is required")
	}
	if backendPort == 0 {
		return fmt.Errorf("--backend-port is required")
	}

	secretKey, err := loadOrGenerateSecretKey()
	if err != nil {
		return fmt.Errorf("failed to load/generate secret key: %w", err)
	}

	// Save proxy config with backend port
	if err := saveProxyConfig(port, backendPort); err != nil {
		return fmt.Errorf("failed to save proxy config: %w", err)
	}

	targetURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", backendPort))
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/login", handleLogin(proxy, backendPort, secretKey))
	mux.HandleFunc("/", handleProxy(proxy, backendPort, secretKey))

	fmt.Printf("Basic auth proxy listening on :%d -> backend :%d\n", port, backendPort)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

func saveProxyConfig(proxyPort, backendPort int) error {
	proxyConfigPath := filepath.Join(configDir, "basic-auth-proxy.json")
	cfg := struct {
		Port        int `json:"port"`
		BackendPort int `json:"backend_port"`
	}{
		Port:        proxyPort,
		BackendPort: backendPort,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(proxyConfigPath, data, 0644)
}

func loadOrGenerateSecretKey() ([]byte, error) {
	configPath := filepath.Join(configDir, configFile)

	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
		key, err := base64.StdEncoding.DecodeString(cfg.SecretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decode secret key: %w", err)
		}
		if len(key) == 32 {
			return key, nil
		}
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}

	cfg := config{SecretKey: base64.StdEncoding.EncodeToString(key)}
	data, err = json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	return key, nil
}

func encryptToken(key []byte, data *tokenData) (string, error) {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func decryptToken(key []byte, encrypted string) (*tokenData, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var data tokenData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func handleLogin(proxy *httputil.ReverseProxy, backendPort int, secretKey []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			serveLoginPage(w, r, "")
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			serveLoginPage(w, r, "Username and password are required")
			return
		}

		valid, err := testBackendAuth(backendPort, req.Username, req.Password)
		if err != nil {
			serveLoginPage(w, r, fmt.Sprintf("Backend error: %v", err))
			return
		}

		if !valid {
			serveLoginPage(w, r, "Invalid username or password")
			return
		}

		token, err := encryptToken(secretKey, &tokenData{
			Username:  req.Username,
			Password:  req.Password,
			CreatedAt: time.Now().Unix(),
		})
		if err != nil {
			http.Error(w, "Failed to create token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(tokenDuration),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func testBackendAuth(backendPort int, username, password string) (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/", backendPort), nil)
	if err != nil {
		return false, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode != http.StatusUnauthorized, nil
}

func handleProxy(proxy *httputil.ReverseProxy, backendPort int, secretKey []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			serveLoginPage(w, r, "")
			return
		}

		data, err := decryptToken(secretKey, cookie.Value)
		if err != nil {
			serveLoginPage(w, r, "")
			return
		}

		if time.Since(time.Unix(data.CreatedAt, 0)) > tokenDuration {
			serveLoginPage(w, r, "Session expired. Please login again.")
			return
		}

		data.CreatedAt = time.Now().Unix()
		newToken, err := encryptToken(secretKey, data)
		if err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    newToken,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(tokenDuration),
			})
		}

		auth := base64.StdEncoding.EncodeToString([]byte(data.Username + ":" + data.Password))
		r.Header.Set("Authorization", "Basic "+auth)

		proxy.ServeHTTP(w, r)
	}
}

func serveLoginPage(w http.ResponseWriter, r *http.Request, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if errMsg != "" {
		errorHTML := strings.ReplaceAll(loginHTML, `<div class="error" id="error"></div>`,
			fmt.Sprintf(`<div class="error show" id="error">%s</div>`, escapeHTML(errMsg)))
		w.Write([]byte(errorHTML))
		return
	}

	w.Write([]byte(loginHTML))
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, `'`, "&#39;")
	return s
}

//go:embed login.html
var loginHTML string
