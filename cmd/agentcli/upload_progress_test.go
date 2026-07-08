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