package agentcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSameGitOrigin_fileURLs(t *testing.T) {
	dir := t.TempDir()
	bare, err := filepath.Abs(filepath.Join(dir, "bare.git"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := normalizeGitOriginURL("file://" + bare)
	if err != nil {
		t.Fatal(err)
	}
	b, err := normalizeGitOriginURL("file://" + filepath.Join(bare, ""))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("%q vs %q", a, b)
	}
	if sameGitOrigin("file://"+bare, "file://"+filepath.Join(dir, "other.git")) {
		t.Fatal("expected different file origins")
	}
}

func TestNormalizeGitOriginURL_roundTrip(t *testing.T) {
	dir := t.TempDir()
	abs, err := filepath.Abs(filepath.Join(dir, "repo.git"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := normalizeGitOriginURL("file://" + abs)
	if err != nil {
		t.Fatal(err)
	}
	want := "file://" + abs
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestProjectBindingsConfigRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	active = RemoteProfile()

	cfg := &agentConfig{
		Default: "http://localhost:8080",
		Domains: []domainConfig{{Server: "http://localhost:8080", Token: "t"}},
	}
	if err := upsertProjectBinding(cfg, "http://localhost:8080/", "/remote/proj", "/local/proj"); err != nil {
		t.Fatal(err)
	}
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.ProjectBindings) != 1 {
		t.Fatalf("bindings: %+v", loaded.ProjectBindings)
	}
	if loaded.ProjectBindings[0].RemoteDir != "/remote/proj" {
		t.Fatalf("remote_dir=%q", loaded.ProjectBindings[0].RemoteDir)
	}

	path, _ := configFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["project_bindings"]; !ok {
		t.Fatal("expected project_bindings in json")
	}
}