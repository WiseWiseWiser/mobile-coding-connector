package fileupload

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	meta := uploadMeta{DestPath: "/tmp/out.bin", TotalChunks: 2, TotalSize: 4, ChmodExec: true}
	if err := saveUploadMeta(dir, meta); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCachedChunk(dir, 0, []byte("ab")); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCachedChunk(dir, 1, []byte("cd")); err != nil {
		t.Fatal(err)
	}
	got, err := listCachedChunkIndices(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("indices=%v", got)
	}
	out := filepath.Join(dir, "out.bin")
	n, err := assembleCachedFile(dir, meta, out)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("size=%d", n)
	}
	data, _ := os.ReadFile(out)
	if string(data) != "abcd" {
		t.Fatalf("data=%q", data)
	}
}

func TestAssembleCachedFile_GunzipsCompressedPayload(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("hello-compressed-upload-payload")
	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	gz := gzBuf.Bytes()
	mid := len(gz) / 2
	if mid == 0 {
		t.Fatal("gzip too small to split")
	}
	meta := uploadMeta{
		DestPath:    filepath.Join(dir, "final.bin"),
		TotalChunks: 2,
		TotalSize:   int64(len(gz)),
		Compressed:  true,
	}
	if err := saveUploadMeta(dir, meta); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCachedChunk(dir, 0, gz[:mid]); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCachedChunk(dir, 1, gz[mid:]); err != nil {
		t.Fatal(err)
	}
	n, err := assembleCachedFile(dir, meta, meta.DestPath)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(raw)) {
		t.Fatalf("size=%d want %d", n, len(raw))
	}
	got, err := os.ReadFile(meta.DestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("got %q want %q", got, raw)
	}
}

func TestIsMultipartBodyTimeout(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{os.ErrDeadlineExceeded, true},
		{&timeoutNetError{}, true},
		{errString("failed to parse form: read tcp 127.0.0.1:1->127.0.0.1:2: i/o timeout"), true},
		{errString("invalid multipart boundary"), false},
	}
	for _, tc := range cases {
		if got := isMultipartBodyTimeout(tc.err); got != tc.want {
			t.Errorf("err=%v: got %v want %v", tc.err, got, tc.want)
		}
	}
}

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "i/o timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

type errString string

func (e errString) Error() string { return string(e) }