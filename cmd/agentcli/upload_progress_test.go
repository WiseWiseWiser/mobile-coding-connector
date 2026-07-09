package agentcli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestPrintUploadProgress_retrying(t *testing.T) {
	out := captureStdout(t, func() {
		printUploadProgress(client.UploadProgress{
			ChunkIndex:  24,
			TotalChunks: 40,
			Attempt:     2,
			MaxAttempts: 5,
			Phase:       client.UploadChunkRetrying,
			Err:         fmt.Errorf("502 Bad Gateway: error code: 502"),
		})
	})
	want := "  chunk 25/40 retrying (attempt 2/5: error code: 502)...\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestPrintUploadProgress_uploadedWithRetries(t *testing.T) {
	out := captureStdout(t, func() {
		printUploadProgress(client.UploadProgress{
			ChunkIndex:     24,
			TotalChunks:    40,
			CompletedBytes: 50 * 1024 * 1024,
			TotalBytes:     78 * 1024 * 1024,
			Attempt:        3,
			MaxAttempts:    5,
			Phase:          client.UploadChunkUploaded,
		})
	})
	if !bytes.Contains([]byte(out), []byte("chunk 25/40 uploaded")) {
		t.Fatalf("missing uploaded line: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte(", 3 attempts")) {
		t.Fatalf("missing attempt suffix: %q", out)
	}
}

func TestPrintUploadDirProgress_fileStart(t *testing.T) {
	out := captureStdout(t, func() {
		printUploadDirProgress(client.UploadDirProgress{
			FileIndex:      1,
			TotalItems:     5,
			RelativePath:   "a.txt",
			Phase:          client.UploadDirPhaseFileStart,
			FileSize:       33,
			CompletedBytes: 0,
			TotalBytes:     12_400_000,
		})
	})
	want := "  [1/5] a.txt (33 B) — 0% overall\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestPrintUploadDirProgress_dirCreated(t *testing.T) {
	out := captureStdout(t, func() {
		printUploadDirProgress(client.UploadDirProgress{
			FileIndex:      4,
			TotalItems:     5,
			RelativePath:   "emptydir/",
			Phase:          client.UploadDirPhaseDirCreated,
			CompletedBytes: 8_064_512,
			TotalBytes:     12_400_000,
		})
	})
	want := "  [4/5] created emptydir/ — 65% overall\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestPrintUploadDirProgress_chunkUploaded(t *testing.T) {
	out := captureStdout(t, func() {
		printUploadDirProgress(client.UploadDirProgress{
			FileIndex:      2,
			TotalItems:     5,
			RelativePath:   "sub/b.txt",
			CompletedBytes: 2_000_000,
			TotalBytes:     12_400_000,
			Chunk: client.UploadProgress{
				ChunkIndex:     0,
				TotalChunks:    4,
				CompletedBytes: 2_000_000,
				TotalBytes:     8_000_000,
				Phase:          client.UploadChunkUploaded,
			},
		})
	})
	want := "    chunk 1/4 uploaded (2.00 MB / 8.00 MB, 25%) — 16% overall\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestPrintUploadDirProgress_chunkRetrying(t *testing.T) {
	out := captureStdout(t, func() {
		printUploadDirProgress(client.UploadDirProgress{
			FileIndex:      2,
			TotalItems:     5,
			RelativePath:   "sub/b.txt",
			CompletedBytes: 2_000_000,
			TotalBytes:     12_400_000,
			Chunk: client.UploadProgress{
				ChunkIndex:  1,
				TotalChunks: 4,
				Attempt:     2,
				MaxAttempts: 5,
				Phase:       client.UploadChunkRetrying,
				Err:         fmt.Errorf("502 Bad Gateway: error code: 502"),
			},
		})
	})
	want := "    chunk 2/4 retrying (attempt 2/5: error code: 502)... — 16% overall\n"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestUploadFailureHint_sessionLost(t *testing.T) {
	hint := uploadFailureHint(fmt.Errorf("upload chunk 28 failed: 404 Not Found: upload session not found"))
	if hint == "" || !bytes.Contains([]byte(hint), []byte("re-run upload")) {
		t.Fatalf("hint = %q", hint)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}