package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRemoteFilePath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/home":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"home":"/home/remote","cwd":"/home/remote"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")

	tests := []struct {
		in   string
		want string
	}{
		{"~/server.log", "/home/remote/server.log"},
		{"server.log", "/home/remote/server.log"},
		{"/tmp/server.log", "/tmp/server.log"},
		{"~", "/home/remote"},
	}

	for _, tc := range tests {
		got, err := c.ResolveRemoteFilePath(tc.in)
		if err != nil {
			t.Fatalf("ResolveRemoteFilePath(%q) error = %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ResolveRemoteFilePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDownloadFile(t *testing.T) {
	const remoteContent = "hello from remote log\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/home":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"home":"/home/remote","cwd":"/home/remote"}`))
		case "/api/files/download":
			if r.URL.Query().Get("path") != "/home/remote/server.log" {
				t.Fatalf("unexpected download path: %q", r.URL.Query().Get("path"))
			}
			w.Write([]byte(remoteContent))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	dir := t.TempDir()
	localPath := filepath.Join(dir, "server.log")

	result, err := c.DownloadFile("~/server.log", localPath, nil)
	if err != nil {
		t.Fatalf("DownloadFile() error = %v", err)
	}
	if result.RemotePath != "/home/remote/server.log" {
		t.Fatalf("RemotePath = %q", result.RemotePath)
	}
	if result.LocalPath != localPath {
		t.Fatalf("LocalPath = %q", result.LocalPath)
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != remoteContent {
		t.Fatalf("file content = %q", string(data))
	}
}

func TestDownloadFileDefaultLocalName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/home":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"home":"/home/remote","cwd":"/home/remote"}`))
		case "/api/files/download":
			w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	dir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWD)

	result, err := c.DownloadFile("~/server.log", "", nil)
	if err != nil {
		t.Fatalf("DownloadFile() error = %v", err)
	}
	if !strings.HasSuffix(result.LocalPath, "server.log") {
		t.Fatalf("LocalPath = %q, want basename server.log", result.LocalPath)
	}
}