package fileupload

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestStreamWS_RoundTripAndResume(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(t.TempDir(), "out.bin")
	mux := http.NewServeMux()
	RegisterAPIForHome(mux, home)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	payload := bytesRepeat([]byte("xyz"), 4000) // 12KiB
	hash := sha256Hex(payload)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/files/upload/ws"

	conn := dialUploadWS(t, wsURL)
	writeJSONMsg(t, conn, streamOpenMsg{
		Type: "open", Path: dest, Size: int64(len(payload)), Hash: hash,
	})
	var ready streamReadyMsg
	readJSONMsg(t, conn, &ready)
	if ready.Type != "ready" || ready.Offset != 0 {
		t.Fatalf("first ready: %+v", ready)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, payload[:1500]); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	conn = dialUploadWS(t, wsURL)
	writeJSONMsg(t, conn, streamOpenMsg{
		Type: "open", Path: dest, Size: int64(len(payload)), Hash: hash,
	})
	readJSONMsg(t, conn, &ready)
	if ready.Type != "ready" {
		t.Fatalf("resume ready type=%s", ready.Type)
	}
	if ready.Offset < 1500 {
		t.Fatalf("resume offset=%d want >=1500", ready.Offset)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, payload[ready.Offset:]); err != nil {
		t.Fatal(err)
	}
	writeJSONMsg(t, conn, map[string]string{"type": "commit"})
	var done streamDoneMsg
	// may see acks first
	waitDone(t, conn, &done)
	if done.Type != "done" {
		t.Fatalf("done=%+v", done)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("dest mismatch len=%d want=%d", len(got), len(payload))
	}
}

func TestStreamWS_HashMismatchOnCommit(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(t.TempDir(), "out.bin")
	mux := http.NewServeMux()
	RegisterAPIForHome(mux, home)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	payload := []byte("hello-stream-upload")
	hash := sha256Hex(payload)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/files/upload/ws"
	conn := dialUploadWS(t, wsURL)
	writeJSONMsg(t, conn, streamOpenMsg{
		Type: "open", Path: dest, Size: int64(len(payload)), Hash: hash,
	})
	var ready streamReadyMsg
	readJSONMsg(t, conn, &ready)
	// send corrupted last byte
	bad := append([]byte{}, payload...)
	bad[len(bad)-1] = 'X'
	if err := conn.WriteMessage(websocket.BinaryMessage, bad); err != nil {
		t.Fatal(err)
	}
	writeJSONMsg(t, conn, map[string]string{"type": "commit"})
	var errMsg streamErrMsg
	waitErr(t, conn, &errMsg)
	if errMsg.Type != "error" || !strings.Contains(errMsg.Message, "hash mismatch") {
		t.Fatalf("err=%+v", errMsg)
	}
}

func dialUploadWS(t *testing.T, wsURL string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func writeJSONMsg(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatal(err)
	}
}

func readJSONMsg(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}

func waitDone(t *testing.T, conn *websocket.Conn, done *streamDoneMsg) {
	t.Helper()
	for i := 0; i < 8; i++ {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var kind struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(data, &kind)
		switch kind.Type {
		case "done":
			if err := json.Unmarshal(data, done); err != nil {
				t.Fatal(err)
			}
			return
		case "error":
			t.Fatalf("got error %s", data)
		case "ack":
			continue
		default:
			t.Fatalf("unexpected %s", data)
		}
	}
	t.Fatal("no done")
}

func waitErr(t *testing.T, conn *websocket.Conn, errMsg *streamErrMsg) {
	t.Helper()
	for i := 0; i < 8; i++ {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var kind struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(data, &kind)
		if kind.Type == "error" {
			if err := json.Unmarshal(data, errMsg); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("no error")
}

// A blocked ack write must not stop the read loop. The payload file keeps
// growing past the 512KB window even while the ack frame cannot be written.
func TestStreamWS_BlockedAckDoesNotStopRead(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(t.TempDir(), "out.bin")
	mux := http.NewServeMux()
	RegisterAPIForHome(mux, home)

	var hold atomic.Bool
	release := make(chan struct{})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(&holdListener{Listener: ln, hold: &hold, release: release}) }()
	t.Cleanup(func() {
		close(release)
		_ = srv.Close()
	})

	payload := bytes.Repeat([]byte("a"), 800*1024)
	hash := sha256Hex(payload)
	wsURL := "ws://" + ln.Addr().String() + "/api/files/upload/ws"
	conn := dialUploadWS(t, wsURL)
	t.Cleanup(func() { conn.Close() })
	writeJSONMsg(t, conn, streamOpenMsg{
		Type: "open", Path: dest, Size: int64(len(payload)), Hash: hash,
	})
	var ready streamReadyMsg
	readJSONMsg(t, conn, &ready)
	if ready.Type != "ready" {
		t.Fatalf("ready: %+v", ready)
	}
	hold.Store(true)

	errc := make(chan error, 1)
	go func() {
		chunk := bytes.Repeat([]byte("a"), 64*1024)
		var sent int
		for sent < len(payload) {
			n := len(chunk)
			if sent+n > len(payload) {
				n = len(payload) - sent
			}
			if werr := conn.WriteMessage(websocket.BinaryMessage, chunk[:n]); werr != nil {
				errc <- werr
				return
			}
			sent += n
		}
		errc <- nil
	}()

	root, err := uploadCacheRootIn(home)
	if err != nil {
		t.Fatal(err)
	}
	payloadPath := filepath.Join(root, hash, "payload")
	deadline := time.Now().Add(5 * time.Second)
	var size int64
	for time.Now().Before(deadline) {
		st, statErr := os.Stat(payloadPath)
		if statErr == nil {
			size = st.Size()
			if size > 512*1024 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if size <= 512*1024 {
		t.Fatalf("payload size = %d, want > 512KB while the ack write is blocked", size)
	}
}

type holdListener struct {
	net.Listener
	hold    *atomic.Bool
	release <-chan struct{}
}

func (l *holdListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &holdConn{Conn: c, hold: l.hold, release: l.release}, nil
}

type holdConn struct {
	net.Conn
	hold    *atomic.Bool
	release <-chan struct{}
}

func (c *holdConn) Write(p []byte) (int, error) {
	if c.hold.Load() && len(p) > 0 {
		op := p[0] & 0x0f
		if op == 0x1 || op == 0x0 {
			<-c.release
		}
	}
	return c.Conn.Write(p)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func bytesRepeat(b []byte, n int) []byte {
	out := make([]byte, 0, len(b)*n)
	for i := 0; i < n; i++ {
		out = append(out, b...)
	}
	return out
}
