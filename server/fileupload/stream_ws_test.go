package fileupload

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
