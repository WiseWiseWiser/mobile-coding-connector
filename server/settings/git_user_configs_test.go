package settings

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
)

func TestGitUserConfigsAPI(t *testing.T) {
	chdirTemp(t)

	mux := http.NewServeMux()
	RegisterAPI(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/settings/git-user-configs", strings.NewReader(`{"id":"work","name":"Jane Doe","email":"jane@example.com"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/settings/git-user-configs", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"work"`) {
		t.Fatalf("GET body missing saved config: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/settings/git-user-configs?id=work", strings.NewReader(`{"name":"Jane Q","email":"janeq@example.com"}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"email":"janeq@example.com"`) {
		t.Fatalf("PATCH body missing updated email: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/settings/git-user-configs?id=work", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, body = %s", rec.Code, rec.Body.String())
	}

	configs, err := LoadGitUserConfigs()
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 0 {
		t.Fatalf("len(configs) = %d, want 0", len(configs))
	}
}

func TestSaveGitUserConfigsNormalizesAndWritesFile(t *testing.T) {
	chdirTemp(t)

	configs, err := SaveGitUserConfigs([]GitUserConfig{
		{Name: "Jane Doe", Email: "jane@example.com"},
		{Name: "Jane Doe", Email: "jane@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 {
		t.Fatalf("len(configs) = %d, want 1", len(configs))
	}
	if configs[0].ID == "" {
		t.Fatal("expected generated id")
	}
	if _, err := os.Stat(config.GitUserConfigsFile); err != nil {
		t.Fatalf("stat %s: %v", config.GitUserConfigsFile, err)
	}
}

func chdirTemp(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(old)
	})
	if err := os.MkdirAll(filepath.Dir(config.GitUserConfigsFile), 0755); err != nil {
		t.Fatal(err)
	}
}
