package client

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeServer captures the multipart form of a conditional write and replies
// with the configured status/body.
type writeServer struct {
	status int
	body   string
	// gotPath/gotExpectedMD5/gotContent record the last request.
	gotPath        string
	gotExpectedMD5 string
	gotContent     string
	// contentType is the raw request Content-Type header.
	contentType string
	sawExpected bool
	requests    int
}

func (ws *writeServer) start(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws.requests++
		ws.contentType = r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(ws.contentType)
		if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
			t.Errorf("unexpected content type %q (err %v)", ws.contentType, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("read part: %v", err)
				break
			}
			data, _ := io.ReadAll(part)
			switch part.FormName() {
			case "path":
				ws.gotPath = string(data)
			case "expected_md5":
				ws.gotExpectedMD5 = string(data)
				ws.sawExpected = true
			case "file":
				ws.gotContent = string(data)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if ws.status != 0 {
			w.WriteHeader(ws.status)
		}
		_, _ = w.Write([]byte(ws.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeLocalFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "staged.txt")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWriteFileConditionalSendsPrecondition(t *testing.T) {
	ws := &writeServer{body: `{"status":"ok","path":"/tmp/a.txt","resolved_path":"/tmp/a.txt","size":5,"md5":"abc","created":false,"mode":"-rw-r--r--"}`}
	srv := ws.start(t)

	c := New(srv.URL, "tok")
	local := writeLocalFile(t, "hello")

	result, err := c.WriteFileConditional("/tmp/a.txt", local, "base-md5")
	if err != nil {
		t.Fatalf("WriteFileConditional error = %v", err)
	}
	if result.MD5 != "abc" || result.Size != 5 || result.Created {
		t.Fatalf("result = %+v", result)
	}
	if ws.gotPath != "/tmp/a.txt" {
		t.Fatalf("path = %q", ws.gotPath)
	}
	if !ws.sawExpected || ws.gotExpectedMD5 != "base-md5" {
		t.Fatalf("expected_md5 = %q (present=%v)", ws.gotExpectedMD5, ws.sawExpected)
	}
	if ws.gotContent != "hello" {
		t.Fatalf("content = %q", ws.gotContent)
	}
}

func TestWriteFileConditionalOmitsEmptyPrecondition(t *testing.T) {
	ws := &writeServer{body: `{"status":"ok","md5":"abc"}`}
	srv := ws.start(t)

	c := New(srv.URL, "")
	local := writeLocalFile(t, "hello")

	if _, err := c.WriteFileConditional("/tmp/a.txt", local, ""); err != nil {
		t.Fatalf("WriteFileConditional error = %v", err)
	}
	if ws.sawExpected {
		t.Fatalf("expected_md5 must be omitted when empty, got %q", ws.gotExpectedMD5)
	}
}

func TestWriteFileConditionalConflict(t *testing.T) {
	ws := &writeServer{
		status: http.StatusConflict,
		body: `{"error":"file changed since it was read","path":"/tmp/a.txt",` +
			`"expected_md5":"base-md5","current_md5":"current-md5","current_exists":true,` +
			`"current_size":12,"current_mod_time":"2026-09-24T10:00:00Z"}`,
	}
	srv := ws.start(t)

	c := New(srv.URL, "")
	local := writeLocalFile(t, "hello")

	_, err := c.WriteFileConditional("/tmp/a.txt", local, "base-md5")
	var conflict *FileConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error = %v (%T), want *FileConflictError", err, err)
	}
	if conflict.ExpectedMD5 != "base-md5" || conflict.CurrentMD5 != "current-md5" {
		t.Fatalf("conflict = %+v", conflict)
	}
	if !conflict.CurrentExists || conflict.CurrentSize != 12 {
		t.Fatalf("conflict = %+v", conflict)
	}
	if conflict.CurrentModTime != "2026-09-24T10:00:00Z" {
		t.Fatalf("mod time = %q", conflict.CurrentModTime)
	}
}

func TestWriteFileConditionalUnsupportedServer(t *testing.T) {
	ws := &writeServer{status: http.StatusNotFound, body: "404 page not found\n"}
	srv := ws.start(t)

	c := New(srv.URL, "")
	local := writeLocalFile(t, "hello")

	_, err := c.WriteFileConditional("/tmp/a.txt", local, "base-md5")
	var unsupported *UnsupportedWriteError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v (%T), want *UnsupportedWriteError", err, err)
	}
	if !strings.Contains(err.Error(), "does not support conditional writes") {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestWriteFileConditionalOtherError(t *testing.T) {
	ws := &writeServer{status: http.StatusBadRequest, body: `{"error":"path is a directory"}`}
	srv := ws.start(t)

	c := New(srv.URL, "")
	local := writeLocalFile(t, "hello")

	_, err := c.WriteFileConditional("/tmp/dir", local, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "path is a directory") {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestWriteFileConditionalMissingLocalFile(t *testing.T) {
	ws := &writeServer{body: `{"status":"ok"}`}
	srv := ws.start(t)

	c := New(srv.URL, "")
	_, err := c.WriteFileConditional("/tmp/a.txt", filepath.Join(t.TempDir(), "absent"), "")
	if err == nil || !strings.Contains(err.Error(), "failed to open local file") {
		t.Fatalf("error = %v", err)
	}
	if ws.requests != 0 {
		t.Fatalf("requests = %d, want 0", ws.requests)
	}
}

// TestDownloadNoResumeReplacesSameSizeLocalFile guards the edit-session base:
// the default resume path returns early when the local size matches the
// remote size, which would keep a stale copy from an earlier run.
func TestDownloadNoResumeReplacesSameSizeLocalFile(t *testing.T) {
	const remote = "remote-bytes"
	const stale = "stale!!bytes" // same length as remote-bytes

	var gotRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/check":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"exists":true,"path":"/tmp/f.txt","size":12,"is_dir":false}`))
		case "/api/files/download":
			gotRange = r.Header.Get("Range")
			_, _ = w.Write([]byte(remote))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	local := filepath.Join(t.TempDir(), "staged.txt")
	if err := os.WriteFile(local, []byte(stale), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := c.DownloadFile("/tmp/f.txt", local, DownloadOptions{NoResume: true}, nil); err != nil {
		t.Fatalf("DownloadFile error = %v", err)
	}
	got, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != remote {
		t.Fatalf("local content = %q, want %q", got, remote)
	}
	if gotRange != "" {
		t.Fatalf("Range = %q, want full download", gotRange)
	}
}

// TestDownloadResumeStillSkipsSameSizeLocalFile documents the default behavior
// that NoResume deliberately bypasses.
func TestDownloadResumeStillSkipsSameSizeLocalFile(t *testing.T) {
	const remote = "remote-bytes"
	const stale = "stale!!bytes"

	downloaded := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/files/check":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"exists":true,"path":"/tmp/f.txt","size":12,"is_dir":false}`))
		case "/api/files/download":
			downloaded = true
			_, _ = w.Write([]byte(remote))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	local := filepath.Join(t.TempDir(), "staged.txt")
	if err := os.WriteFile(local, []byte(stale), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := c.DownloadFile("/tmp/f.txt", local, DownloadOptions{}, nil); err != nil {
		t.Fatalf("DownloadFile error = %v", err)
	}
	if downloaded {
		t.Fatal("expected resume path to skip the GET")
	}
	got, _ := os.ReadFile(local)
	if string(got) != stale {
		t.Fatalf("local content = %q, want stale copy kept", got)
	}
}
