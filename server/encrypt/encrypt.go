package encrypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

const (
	defaultPrivateKeyFile = ".ai-critic-enc-key"
)

var (
	rsaPrivateKey *rsa.PrivateKey
	rsaPublicPEM  string // PEM-encoded SPKI public key for the frontend
	loadOnce      sync.Once
	loadErr       error
	keysMu        sync.Mutex // protects key generation and reload

	keyFileMu      sync.RWMutex
	privateKeyFile = defaultPrivateKeyFile
)

// SetKeyFile sets the path to the encryption private key file.
// The public key file is derived by appending ".pub" to the path.
// The provided path has any ".pub" suffix stripped before use.
// Must be called before the server starts.
func SetKeyFile(path string) {
	path = strings.TrimSuffix(path, ".pub")
	keyFileMu.Lock()
	defer keyFileMu.Unlock()
	privateKeyFile = path
}

func getPrivateKeyFile() string {
	keyFileMu.RLock()
	defer keyFileMu.RUnlock()
	return privateKeyFile
}

func getPublicKeyFile() string {
	return getPrivateKeyFile() + ".pub"
}

// Available returns true if the encryption key pair is loaded and ready
func Available() bool {
	loadKeys()
	return rsaPrivateKey != nil
}

// loadKeys loads the RSA key pair from disk.
// If the key files don't exist, the keys remain nil (encryption not available).
func loadKeys() {
	loadOnce.Do(func() {
		privFile := getPrivateKeyFile()

		// Check if key files exist
		if _, err := os.Stat(privFile); os.IsNotExist(err) {
			// Keys not generated - encryption not available
			return
		}

		// Read private key
		privData, err := os.ReadFile(privFile)
		if err != nil {
			loadErr = fmt.Errorf("failed to read private key file %s: %w", privFile, err)
			return
		}

		// Parse OpenSSH private key
		rawKey, err := ssh.ParseRawPrivateKey(privData)
		if err != nil {
			loadErr = fmt.Errorf("failed to parse private key: %w", err)
			return
		}

		rsaKey, ok := rawKey.(*rsa.PrivateKey)
		if !ok {
			loadErr = fmt.Errorf("private key is not RSA (got %T)", rawKey)
			return
		}
		rsaPrivateKey = rsaKey

		// Generate SPKI PEM public key for the frontend
		pubKeyBytes, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
		if err != nil {
			loadErr = fmt.Errorf("failed to marshal public key: %w", err)
			return
		}

		pubPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyBytes,
		})
		rsaPublicPEM = string(pubPEM)
	})
}

// Decrypt decrypts data that was encrypted with the public key using RSA-OAEP with SHA-256.
// The input is expected to be base64-encoded chunks separated by "." (since RSA can only
// encrypt data smaller than the key size, we split into chunks on the frontend).
func Decrypt(encryptedBase64 string) (string, error) {
	loadKeys()
	if rsaPrivateKey == nil {
		if loadErr != nil {
			return "", loadErr
		}
		return "", fmt.Errorf("encryption keys not available, run: go run ./script/crypto/gen")
	}

	// Split by "." for chunked encryption
	chunks := strings.Split(encryptedBase64, ".")
	var result []byte
	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}
		ciphertext, err := base64.StdEncoding.DecodeString(chunk)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 chunk: %w", err)
		}

		plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivateKey, ciphertext, nil)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt chunk: %w", err)
		}
		result = append(result, plaintext...)
	}
	return string(result), nil
}

const keyBits = 3072

// reloadKeys forces a reload of the key pair from disk.
// Must be called with keysMu held.
func reloadKeys() {
	rsaPrivateKey = nil
	rsaPublicPEM = ""
	loadErr = nil
	loadOnce = sync.Once{}
	loadKeys()
}

// GenerateKeys generates a new RSA key pair and writes it to disk.
// If the key files already exist, they are overwritten.
func GenerateKeys() error {
	keysMu.Lock()
	defer keysMu.Unlock()

	privateKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %v", err)
	}

	// Write private key in OpenSSH format
	privBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return fmt.Errorf("failed to marshal private key to OpenSSH format: %v", err)
	}
	privPEM := pem.EncodeToMemory(privBlock)
	if err := os.WriteFile(getPrivateKeyFile(), privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key file: %v", err)
	}

	// Write public key in OpenSSH format
	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create SSH public key: %v", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(pubKey)
	if err := os.WriteFile(getPublicKeyFile(), pubBytes, 0644); err != nil {
		return fmt.Errorf("failed to write public key file: %v", err)
	}

	// Reload keys from disk
	reloadKeys()
	return nil
}

// KeyStatus represents the encryption key pair status
type KeyStatus struct {
	Exists         bool   `json:"exists"`
	Valid          bool   `json:"valid"`
	Error          string `json:"error,omitempty"`
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
}

// GetKeyStatus returns the current key pair status, including validation
func GetKeyStatus() KeyStatus {
	privFile := getPrivateKeyFile()
	pubFile := getPublicKeyFile()

	status := KeyStatus{
		PrivateKeyPath: privFile,
		PublicKeyPath:  pubFile,
	}

	// Check private key
	privData, err := os.ReadFile(privFile)
	if err != nil {
		if os.IsNotExist(err) {
			status.Error = "private key file does not exist"
		} else {
			status.Error = fmt.Sprintf("failed to read private key: %v", err)
		}
		return status
	}
	if len(strings.TrimSpace(string(privData))) == 0 {
		status.Error = "private key file is empty"
		return status
	}
	status.Exists = true

	// Validate private key format
	rawKey, err := ssh.ParseRawPrivateKey(privData)
	if err != nil {
		status.Error = fmt.Sprintf("invalid private key format: %v", err)
		return status
	}
	rsaKey, ok := rawKey.(*rsa.PrivateKey)
	if !ok {
		status.Error = fmt.Sprintf("private key is not RSA (got %T)", rawKey)
		return status
	}

	// Check public key
	pubData, err := os.ReadFile(pubFile)
	if err != nil {
		if os.IsNotExist(err) {
			status.Error = "public key file does not exist"
		} else {
			status.Error = fmt.Sprintf("failed to read public key: %v", err)
		}
		return status
	}
	if len(strings.TrimSpace(string(pubData))) == 0 {
		status.Error = "public key file is empty"
		return status
	}

	// Validate public key format and match with private key
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubData)
	if err != nil {
		status.Error = fmt.Sprintf("invalid public key format: %v", err)
		return status
	}

	// Verify the public key matches the private key
	expectedPub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		status.Error = fmt.Sprintf("failed to derive public key from private key: %v", err)
		return status
	}
	if string(pubKey.Marshal()) != string(expectedPub.Marshal()) {
		status.Error = "public key does not match private key"
		return status
	}

	status.Valid = true
	return status
}

// RegisterAPI registers the encryption-related endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/encrypt/public-key", handlePublicKey)
	mux.HandleFunc("/api/encrypt/status", handleStatus)
	mux.HandleFunc("/api/encrypt/generate", handleGenerate)
}

func handlePublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	loadKeys()

	// If keys aren't available, return empty public_key so frontend knows
	if rsaPrivateKey == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"public_key": ""})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"public_key": rsaPublicPEM})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetKeyStatus())
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := GenerateKeys(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetKeyStatus())
}
