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
	privateKeyFile = ".ai-critic-enc-key"
	publicKeyFile  = ".ai-critic-enc-key.pub"
)

var (
	rsaPrivateKey *rsa.PrivateKey
	rsaPublicPEM  string // PEM-encoded SPKI public key for the frontend
	loadOnce      sync.Once
	loadErr       error
)

// Available returns true if the encryption key pair is loaded and ready
func Available() bool {
	loadKeys()
	return rsaPrivateKey != nil
}

// loadKeys loads the RSA key pair from disk.
// If the key files don't exist, the keys remain nil (encryption not available).
func loadKeys() {
	loadOnce.Do(func() {
		// Check if key files exist
		if _, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
			// Keys not generated - encryption not available
			return
		}

		// Read private key
		privData, err := os.ReadFile(privateKeyFile)
		if err != nil {
			loadErr = fmt.Errorf("failed to read private key file %s: %w", privateKeyFile, err)
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

// RegisterAPI registers the public key endpoint
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/encrypt/public-key", handlePublicKey)
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
