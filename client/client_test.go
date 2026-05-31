package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthStatus_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"status":      "ok",
			"initialized": true,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	result, err := c.AuthStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatal("expected OK=true")
	}
	if !result.Initialized {
		t.Fatal("expected Initialized=true")
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
}

func TestAuthStatus_NotInitialized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"status":      "not_initialized",
			"initialized": false,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	result, err := c.AuthStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected OK=false")
	}
	if result.Initialized {
		t.Fatal("expected Initialized=false")
	}
}

func TestAuthStatus_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"status":      "unauthorized",
			"initialized": true,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	result, err := c.AuthStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected OK=false")
	}
	if !result.Initialized {
		t.Fatal("expected Initialized=true")
	}
}

func TestAuthStatus_ConnecionRefused(t *testing.T) {
	c := New("http://127.0.0.1:1", "")
	_, err := c.AuthStatus()
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}
