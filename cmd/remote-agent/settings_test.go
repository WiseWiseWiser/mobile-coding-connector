package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

func TestRunSettingsGitUsersList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/settings/git-user-configs" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"work","name":"Jane Doe","email":"jane@example.com","createdAt":"2026-05-07T00:00:00Z"}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	var runErr error
	withStdout(&out, func() {
		runErr = runSettings(func() (*client.Client, error) {
			return client.New(server.URL, ""), nil
		}, []string{"git-users", "list"})
	})
	if runErr != nil {
		t.Fatalf("runSettings() error = %v", runErr)
	}
	if !strings.Contains(out.String(), "Jane Doe") || !strings.Contains(out.String(), "jane@example.com") {
		t.Fatalf("output missing identity: %s", out.String())
	}
}

func withStdout(out *bytes.Buffer, fn func()) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()
	_ = w.Close()
	_, _ = io.Copy(out, r)
	_ = r.Close()
}
