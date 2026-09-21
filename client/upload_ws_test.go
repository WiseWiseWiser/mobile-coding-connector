package client

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhd2015/ai-critic/server/fileupload"
)

func TestUploadFileWS_GzipRoundTrip(t *testing.T) {
	home := t.TempDir()
	destDir := t.TempDir()
	mux := http.NewServeMux()
	fileupload.RegisterAPIForHome(mux, home)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	src := filepath.Join(t.TempDir(), "plain.txt")
	raw := bytes.Repeat([]byte("aaaaaaaa"), 8*1024)
	if err := os.WriteFile(src, raw, 0644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(destDir, "plain.txt")
	cli := New(srv.URL, "")
	res, err := cli.UploadFile(src, dest, UploadOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("dest len=%d want %d", len(got), len(raw))
	}
}

func TestUploadFileWS_NoCompressAndChmod(t *testing.T) {
	home := t.TempDir()
	destDir := t.TempDir()
	mux := http.NewServeMux()
	fileupload.RegisterAPIForHome(mux, home)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	src := filepath.Join(t.TempDir(), "bin")
	raw := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := os.WriteFile(src, raw, 0755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(destDir, "bin")
	cli := New(srv.URL, "")
	res, err := cli.UploadFile(src, dest, UploadOptions{NoCompress: true, ChmodExec: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("data=%v", got)
	}
	st, err := os.Stat(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&0o111 == 0 {
		t.Fatalf("mode=%s want executable", st.Mode())
	}
}

func TestIsUploadWSUnsupported_BadHandshake(t *testing.T) {
	if !isUploadWSUnsupported(fmt.Errorf("upload websocket dial: websocket: bad handshake")) {
		t.Fatal("bad handshake should fall back to HTTP chunks")
	}
	if isUploadWSUnsupported(fmt.Errorf("connection refused")) {
		t.Fatal("connection refused is not unsupported")
	}
}

func TestPrepareUploadPayloadFile_CompressWhenSmaller(t *testing.T) {
	src := filepath.Join(t.TempDir(), "plain.txt")
	raw := bytes.Repeat([]byte("aaaaaaaa"), 8*1024)
	if err := os.WriteFile(src, raw, 0644); err != nil {
		t.Fatal(err)
	}
	path, hash, wire, compressed, cleanup, err := prepareUploadPayloadFile(src, false)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if !compressed {
		t.Fatal("expected gzip")
	}
	if wire >= int64(len(raw)) {
		t.Fatalf("wire=%d raw=%d", wire, len(raw))
	}
	if hash == "" || path == src {
		t.Fatalf("hash=%q path=%s", hash, path)
	}
}
