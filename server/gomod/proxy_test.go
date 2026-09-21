package gomod

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	toml := filepath.Join(root, "github.com", "!burnt!sushi", "toml", "@v")
	if err := os.MkdirAll(toml, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(toml, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("v1.6.0.info", `{"Version":"v1.6.0","Time":"2025-12-18T12:15:22Z"}`)
	write("v1.6.0.mod", "module github.com/BurntSushi/toml\n")
	write("v1.6.0.zip", "ZIPBYTES-v1.6.0")
	write("v0.3.1.mod", "module github.com/BurntSushi/toml\n")
	xmod := filepath.Join(root, "golang.org", "x", "mod", "@v")
	if err := os.MkdirAll(xmod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xmod, "v0.20.0.mod"), []byte("module golang.org/x/mod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func get(t *testing.T, h http.Handler, path string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w, w.Body.String()
}

func TestProxyList(t *testing.T) {
	h := NewProxyHandler(newTestRoot(t))
	w, body := get(t, h, "/github.com/!burnt!sushi/toml/@v/list")
	if w.Code != 200 {
		t.Fatalf("list: got %d want 200", w.Code)
	}
	if body != "v0.3.1\nv1.6.0\n" {
		t.Fatalf("list: got %q", body)
	}
}

func TestProxyLatestFromInfo(t *testing.T) {
	h := NewProxyHandler(newTestRoot(t))
	w, body := get(t, h, "/github.com/!burnt!sushi/toml/@latest")
	if w.Code != 200 {
		t.Fatalf("latest: got %d want 200", w.Code)
	}
	var info struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal([]byte(body), &info); err != nil {
		t.Fatalf("latest not json: %v (%q)", err, body)
	}
	if info.Version != "v1.6.0" {
		t.Fatalf("latest version: got %q", info.Version)
	}
}

func TestProxySynthesizeInfo(t *testing.T) {
	h := NewProxyHandler(newTestRoot(t))
	w, body := get(t, h, "/golang.org/x/mod/@v/v0.20.0.info")
	if w.Code != 200 {
		t.Fatalf("synth info: got %d want 200", w.Code)
	}
	var info struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal([]byte(body), &info); err != nil {
		t.Fatalf("synth info not json: %v (%q)", err, body)
	}
	if info.Version != "v0.20.0" {
		t.Fatalf("synth info version: got %q", info.Version)
	}
}

func TestProxyZipBytes(t *testing.T) {
	h := NewProxyHandler(newTestRoot(t))
	w, body := get(t, h, "/github.com/!burnt!sushi/toml/@v/v1.6.0.zip")
	if w.Code != 200 || body != "ZIPBYTES-v1.6.0" {
		t.Fatalf("zip: got %d %q", w.Code, body)
	}
}

func TestProxyMissingModule404(t *testing.T) {
	h := NewProxyHandler(newTestRoot(t))
	w, _ := get(t, h, "/example.com/nonexistent/@v/v1.0.0.info")
	if w.Code != 404 {
		t.Fatalf("missing module: got %d want 404", w.Code)
	}
}

func TestProxyMissingInfoNoModNoZip404(t *testing.T) {
	h := NewProxyHandler(newTestRoot(t))
	w, _ := get(t, h, "/github.com/!burnt!sushi/toml/@v/v9.9.9.info")
	if w.Code != 404 {
		t.Fatalf("missing info without mod/zip: got %d want 404", w.Code)
	}
}

func TestProxyDotDotCleanedToRoot404(t *testing.T) {
	h := NewProxyHandler(newTestRoot(t))
	// After path.Clean, ".." components collapse; a leftover empty path 404s.
	w, _ := get(t, h, "/")
	if w.Code != 404 {
		t.Fatalf("empty path: got %d want 404", w.Code)
	}
}
