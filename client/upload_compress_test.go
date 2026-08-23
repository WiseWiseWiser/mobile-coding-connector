package client

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGzipBytesDeterministic_StableAcrossCalls(t *testing.T) {
	src := bytes.Repeat([]byte("hello-upload-compress-"), 1000)
	a, err := gzipBytesDeterministic(src)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond) // wall clock must not affect hash
	b, err := gzipBytesDeterministic(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("gzip output differed across calls (%d vs %d bytes)", len(a), len(b))
	}
	ha, _ := computeBytesChunkPlan(a, ChunkSize)
	hb, _ := computeBytesChunkPlan(b, ChunkSize)
	if ha != hb {
		t.Fatalf("file hash differed: %s vs %s", ha, hb)
	}
}

func TestPrepareUploadPayload_CompressesWhenSmaller(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.txt")
	// Highly compressible payload.
	raw := bytes.Repeat([]byte("aaaaaaaa"), 8*1024)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	payload, compressed, err := prepareUploadPayload(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if !compressed {
		t.Fatal("expected compressed=true for repetitive text")
	}
	if len(payload) >= len(raw) {
		t.Fatalf("compressed size %d not smaller than raw %d", len(payload), len(raw))
	}
	gr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	out := new(bytes.Buffer)
	if _, err := out.ReadFrom(gr); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), raw) {
		t.Fatal("gunzip did not round-trip original bytes")
	}
}

func TestPrepareUploadPayload_FallsBackWhenNotSmaller(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noise.bin")
	// Pre-gzipped (or otherwise high-entropy) bytes should not shrink further.
	seed := bytes.Repeat([]byte("seed-payload-for-noise-"), 256)
	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write(seed); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := gzBuf.Bytes()
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	payload, compressed, err := prepareUploadPayload(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if compressed {
		t.Fatalf("expected fallback to raw for already-compressed data (payload %d, raw %d)", len(payload), len(raw))
	}
	if !bytes.Equal(payload, raw) {
		t.Fatal("expected raw payload when not compressing")
	}
}

func TestPrepareUploadPayload_NoCompress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.txt")
	raw := bytes.Repeat([]byte("bbbbbbbb"), 8*1024)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	payload, compressed, err := prepareUploadPayload(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if compressed {
		t.Fatal("NoCompress should force compressed=false")
	}
	if !bytes.Equal(payload, raw) {
		t.Fatal("NoCompress should upload raw bytes")
	}
}
