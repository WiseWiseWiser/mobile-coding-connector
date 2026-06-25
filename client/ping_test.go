package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPing_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte("pong"))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	result, err := c.Ping()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Latency <= 0 {
		t.Fatalf("latency = %v, want > 0", result.Latency)
	}
}

func TestPing_UnexpectedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("nope"))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.Ping()
	if err == nil || !strings.Contains(err.Error(), "unexpected response") {
		t.Fatalf("expected unexpected response error, got %v", err)
	}
}

func TestPing_ConnectionRefused(t *testing.T) {
	c := New("http://127.0.0.1:1", "")
	_, err := c.Ping()
	if err == nil {
		t.Fatal("expected connection error")
	}
}