package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// checkServer records the decoded /api/files/check request body and replies
// with the configured body.
type checkServer struct {
	status  int
	body    string
	gotBody map[string]any
	raw     string
	calls   int
}

func (cs *checkServer) start(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/files/check" {
			http.NotFound(w, r)
			return
		}
		cs.calls++
		data, _ := io.ReadAll(r.Body)
		cs.raw = string(data)
		_ = json.Unmarshal(data, &cs.gotBody)
		w.Header().Set("Content-Type", "application/json")
		if cs.status != 0 {
			w.WriteHeader(cs.status)
		}
		_, _ = w.Write([]byte(cs.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCheckPathDoesNotRequestDigest(t *testing.T) {
	cs := &checkServer{body: `{"exists":true,"path":"/tmp/a.txt","size":4,"is_dir":false}`}
	srv := cs.start(t)

	info, err := New(srv.URL, "").CheckPath("/tmp/a.txt")
	if err != nil {
		t.Fatalf("CheckPath error = %v", err)
	}
	if info.Size != 4 || !info.Exists {
		t.Fatalf("info = %+v", info)
	}
	if info.MD5 != "" {
		t.Fatalf("md5 = %q, want empty", info.MD5)
	}
	if _, ok := cs.gotBody["md5"]; ok {
		t.Fatalf("plain CheckPath must not ask for md5: %s", cs.raw)
	}
}

func TestCheckPathMD5RequestsAndParsesDigest(t *testing.T) {
	const digest = "8f14e45fceea167a5a36dedd4bea2543"
	cs := &checkServer{body: `{"exists":true,"path":"/tmp/a.txt","size":4,"is_dir":false,"md5":"` + digest + `"}`}
	srv := cs.start(t)

	info, err := New(srv.URL, "").CheckPathMD5("/tmp/a.txt")
	if err != nil {
		t.Fatalf("CheckPathMD5 error = %v", err)
	}
	if info.MD5 != digest {
		t.Fatalf("md5 = %q, want %q", info.MD5, digest)
	}
	if got, ok := cs.gotBody["md5"].(bool); !ok || !got {
		t.Fatalf("request must set md5=true: %s", cs.raw)
	}
	if cs.gotBody["path"] != "/tmp/a.txt" {
		t.Fatalf("request path = %v", cs.gotBody["path"])
	}
}

// TestCheckPathMD5ToleratesOldServer covers a server build that predates digest
// reporting: it ignores the unknown field and returns no md5.
func TestCheckPathMD5ToleratesOldServer(t *testing.T) {
	cs := &checkServer{body: `{"exists":true,"path":"/tmp/a.txt","size":4,"is_dir":false}`}
	srv := cs.start(t)

	info, err := New(srv.URL, "").CheckPathMD5("/tmp/a.txt")
	if err != nil {
		t.Fatalf("CheckPathMD5 error = %v", err)
	}
	if info.MD5 != "" {
		t.Fatalf("md5 = %q, want empty for an old server", info.MD5)
	}
	if !info.Exists || info.Size != 4 {
		t.Fatalf("info = %+v", info)
	}
}

func TestCheckPathMD5MissingPathHasNoDigest(t *testing.T) {
	cs := &checkServer{body: `{"exists":false,"path":"/tmp/absent"}`}
	srv := cs.start(t)

	info, err := New(srv.URL, "").CheckPathMD5("/tmp/absent")
	if err != nil {
		t.Fatalf("CheckPathMD5 error = %v", err)
	}
	if info.Exists {
		t.Fatal("exists = true, want false")
	}
	if info.MD5 != "" {
		t.Fatalf("md5 = %q, want empty", info.MD5)
	}
}

func TestCheckPathMD5PropagatesHTTPError(t *testing.T) {
	cs := &checkServer{status: http.StatusInternalServerError, body: `{"error":"failed to hash file: permission denied"}`}
	srv := cs.start(t)

	_, err := New(srv.URL, "").CheckPathMD5("/tmp/secret")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to hash file") {
		t.Fatalf("error = %v", err)
	}
}
