package agentcli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestRunGitSubcommandsForwardGitToken(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []struct {
			Path string
			Body map[string]any
		}
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		mu.Lock()
		requests = append(requests, struct {
			Path string
			Body map[string]any
		}{Path: r.URL.Path, Body: body})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"type":"exit","code":0}`)
	}))
	defer server.Close()

	resolve := func() (*client.Client, error) {
		return client.New(server.URL, ""), nil
	}

	cases := []struct {
		name     string
		args     []string
		path     string
		gitToken string
	}{
		{
			name:     "clone",
			args:     []string{"clone", "/remote/src", "/remote/dst", "--git-token", "clone-token"},
			path:     "/api/remote-agent/git/clone",
			gitToken: "clone-token",
		},
		{
			name:     "fetch",
			args:     []string{"-C", "/remote/repo", "fetch", "--git-token", "fetch-token"},
			path:     "/api/remote-agent/git/fetch",
			gitToken: "fetch-token",
		},
		{
			name:     "pull",
			args:     []string{"-C", "/remote/repo", "pull", "--git-token", "pull-token"},
			path:     "/api/remote-agent/git/pull",
			gitToken: "pull-token",
		},
		{
			name:     "push",
			args:     []string{"-C", "/remote/repo", "push", "--git-token", "push-token"},
			path:     "/api/remote-agent/git/push",
			gitToken: "push-token",
		},
		{
			name:     "status",
			args:     []string{"-C", "/remote/repo", "status", "--git-token", "status-token"},
			path:     "/api/remote-agent/git/run",
			gitToken: "status-token",
		},
	}

	for i, tc := range cases {
		if err := runGit(resolve, tc.args); err != nil {
			t.Fatalf("%s: runGit() error = %v", tc.name, err)
		}
		mu.Lock()
		requestCount := len(requests)
		var req struct {
			Path string
			Body map[string]any
		}
		if requestCount > i {
			req = requests[i]
		}
		mu.Unlock()
		if requestCount != i+1 {
			t.Fatalf("%s: request count = %d, want %d", tc.name, requestCount, i+1)
		}
		if req.Path != tc.path {
			t.Fatalf("%s: path = %q, want %q", tc.name, req.Path, tc.path)
		}
		if got, _ := req.Body["token"].(string); got != tc.gitToken {
			t.Fatalf("%s: git token = %q, want %q", tc.name, got, tc.gitToken)
		}
	}
}
