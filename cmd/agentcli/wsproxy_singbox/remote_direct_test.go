package wsproxy_singbox

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestPushRemoteDirectOK(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/ws-proxy/remote-direct" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"patterns": []string{"*.db.internal:6606"},
		})
	}))
	defer srv.Close()

	patterns, err := ParseAlsoProxyPatterns([]string{"*.db.internal:6606"})
	if err != nil {
		t.Fatal(err)
	}
	err = pushRemoteDirect(func() (*client.Client, error) {
		return client.New(srv.URL, ""), nil
	}, patterns)
	if err != nil {
		t.Fatalf("pushRemoteDirect: %v", err)
	}
	if !strings.Contains(string(gotBody), "*.db.internal:6606") {
		t.Fatalf("body=%s", gotBody)
	}
}

func TestPushRemoteDirectUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	patterns, _ := ParseAlsoProxyPatterns([]string{"db.example.com:6606"})
	err := pushRemoteDirect(func() (*client.Client, error) {
		return client.New(srv.URL, ""), nil
	}, patterns)
	if err == nil || !strings.Contains(err.Error(), "does not support --remote-direct") {
		t.Fatalf("err=%v", err)
	}
}
