package client

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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

func TestFrameForRateStaysSmallOnSlowLink(t *testing.T) {
	const rate = 40 * 1024
	// 40KB/s can finish 256KB in 6.4s, which is inside the 10s budget, but
	// 512KB would take 12.8s. The frame must not grow into that.
	if got := frameForRate(wsInitFrame, rate); got != wsInitFrame*2 && got != wsInitFrame {
		t.Fatalf("first grow = %d", got)
	}
	cur := wsInitFrame
	for i := 0; i < 8; i++ {
		cur = frameForRate(cur, rate)
	}
	if cur > 256*1024 {
		t.Fatalf("frame grew to %d at 40KB/s; want <= 256KB", cur)
	}
	if cur < wsInitFrame {
		t.Fatalf("frame = %d", cur)
	}
}

func TestWriteDeadlineForFloorRate(t *testing.T) {
	if got := writeDeadlineFor(64 * 1024); got != 8*time.Second {
		t.Fatalf("64KB deadline = %s, want 8s", got)
	}
	if got := writeDeadlineFor(1 << 20); got != 128*time.Second {
		t.Fatalf("1MB deadline = %s, want 128s", got)
	}
}

func TestUploadWriteDeadlineFiresWhenStalled(t *testing.T) {
	done := make(chan struct{})
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Read nothing. The client write blocks once the TCP window fills.
		<-done
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	defer close(done)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	started := time.Now()
	payload := bytes.Repeat([]byte("x"), 64*1024)
	var writeErr error
	for i := 0; i < 64; i++ {
		writeErr = writeWSBinary(conn, payload)
		if writeErr != nil {
			break
		}
	}
	if writeErr == nil {
		t.Fatal("stalled write did not fail")
	}
	elapsed := time.Since(started)
	// The blocked frame is 64KB, so the floor-rate deadline is 8s. Allow
	// the kernel buffer to fill before that deadline starts.
	if elapsed > 30*time.Second {
		t.Fatalf("stalled write took %s, want the 8s frame deadline", elapsed)
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
