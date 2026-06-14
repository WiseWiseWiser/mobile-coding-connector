package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleAuthStatus_Success(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials")
	SetCredentialsFile(credFile)
	os.WriteFile(credFile, []byte("valid-token\n"), 0600)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handleAuthStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "ok" {
		t.Fatalf("status = %v, want ok", result["status"])
	}
	if result["initialized"] != true {
		t.Fatalf("initialized = %v, want true", result["initialized"])
	}
}

func TestHandleAuthStatus_NotInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "nonexistent", "credentials")
	SetCredentialsFile(credFile)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	w := httptest.NewRecorder()
	handleAuthStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "not_initialized" {
		t.Fatalf("status = %v, want not_initialized", result["status"])
	}
	if result["initialized"] != false {
		t.Fatalf("initialized = %v, want false", result["initialized"])
	}
}

func TestHandleAuthStatus_Unauthorized(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials")
	SetCredentialsFile(credFile)
	os.WriteFile(credFile, []byte("good-token\n"), 0600)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()
	handleAuthStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "unauthorized" {
		t.Fatalf("status = %v, want unauthorized", result["status"])
	}
	if result["initialized"] != true {
		t.Fatalf("initialized = %v, want true", result["initialized"])
	}
}

func TestHandleAuthStatus_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/status", nil)
	w := httptest.NewRecorder()
	handleAuthStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}
