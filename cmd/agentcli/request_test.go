package agentcli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestRunRequestGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RequestURI() != "/api/services?project_dir=/tmp/repo" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer server-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	var out bytes.Buffer
	var runErr error
	withStdout(&out, func() {
		runErr = runRequest(func() (*client.Client, error) {
			return client.New(server.URL, "server-token"), nil
		}, []string{"/api/services?project_dir=/tmp/repo"})
	})
	if runErr != nil {
		t.Fatalf("runRequest() error = %v", runErr)
	}
	if strings.TrimSpace(out.String()) != `{"ok":true}` {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunRequestPOSTJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/services/start" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if strings.TrimSpace(string(data)) != `{"id":"svc-123"}` {
			t.Fatalf("request body = %q", string(data))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := runRequest(func() (*client.Client, error) {
		return client.New(server.URL, ""), nil
	}, []string{"/api/services/start", `{"id":"svc-123"}`}); err != nil {
		t.Fatalf("runRequest() error = %v", err)
	}
}

func TestRunRequestPOSTJSONFromPipedStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/some" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if strings.TrimSpace(string(data)) != `{"x":1}` {
			t.Fatalf("request body = %q", string(data))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	withStdinPipe(t, `{"x":1}`, func() {
		if err := runRequest(func() (*client.Client, error) {
			return client.New(server.URL, ""), nil
		}, []string{"/api/some"}); err != nil {
			t.Fatalf("runRequest() error = %v", err)
		}
	})
}

func TestRunRequestRejectsInvalidJSON(t *testing.T) {
	err := runRequest(func() (*client.Client, error) {
		t.Fatalf("resolve should not be called")
		return nil, nil
	}, []string{"/api/services/start", `{bad`})
	if err == nil || !strings.Contains(err.Error(), "json-body must be valid JSON") {
		t.Fatalf("runRequest() error = %v, want invalid JSON error", err)
	}
}

func withStdinPipe(t *testing.T, input string, fn func()) {
	t.Helper()
	old := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := io.WriteString(w, input); err != nil {
		t.Fatalf("write stdin pipe: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close stdin pipe writer: %v", err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = old
		_ = r.Close()
	}()

	fn()
}
