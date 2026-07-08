package fileupload

import (
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