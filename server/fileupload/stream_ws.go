package fileupload

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	streamAckEvery   = 256 * 1024
	streamFsyncEvery = 4 * 1024 * 1024
	streamReadIdle   = 60 * time.Second
	streamPingEvery  = 20 * time.Second
)

var streamUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	hashLockMu sync.Mutex
	hashLocks  = map[string]*sync.Mutex{}
)

func lockFileHash(h string) func() {
	hashLockMu.Lock()
	m, ok := hashLocks[h]
	if !ok {
		m = &sync.Mutex{}
		hashLocks[h] = m
	}
	hashLockMu.Unlock()
	m.Lock()
	return func() { m.Unlock() }
}

type streamOpenMsg struct {
	Type       string `json:"type"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
	Compressed bool   `json:"compressed"`
	ChmodExec  bool   `json:"chmod_exec"`
}

type streamReadyMsg struct {
	Type   string `json:"type"`
	Offset int64  `json:"offset"`
}

type streamAckMsg struct {
	Type   string `json:"type"`
	Offset int64  `json:"offset"`
}

type streamDoneMsg struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

type streamErrMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func handleUploadWS(w http.ResponseWriter, r *http.Request, home string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	conn, err := streamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(streamReadIdle))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(streamReadIdle))
	})

	mt, data, err := conn.ReadMessage()
	if err != nil {
		return
	}
	if mt != websocket.TextMessage {
		writeStreamErr(conn, "expected text open message")
		return
	}
	var open streamOpenMsg
	if err := json.Unmarshal(data, &open); err != nil || open.Type != "open" {
		writeStreamErr(conn, "invalid open message")
		return
	}
	if open.Path == "" || open.Size < 0 || !isFileHash(open.Hash) {
		writeStreamErr(conn, "path, size, and sha256 hash are required")
		return
	}
	destPath := filepath.Clean(open.Path)

	unlock := lockFileHash(open.Hash)
	defer unlock()

	dir, err := uploadCacheDirIn(home, open.Hash)
	if err != nil {
		writeStreamErr(conn, "failed to prepare cache: "+err.Error())
		return
	}
	payloadPath := filepath.Join(dir, "payload")
	offset, err := prepareStreamPayload(dir, payloadPath, destPath, open)
	if err != nil {
		writeStreamErr(conn, err.Error())
		return
	}
	if err := writeStreamJSON(conn, streamReadyMsg{Type: "ready", Offset: offset}); err != nil {
		return
	}

	f, err := os.OpenFile(payloadPath, os.O_WRONLY, 0644)
	if err != nil {
		writeStreamErr(conn, "open payload: "+err.Error())
		return
	}
	payloadClosed := false
	defer func() {
		if !payloadClosed {
			_ = f.Close()
		}
	}()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		writeStreamErr(conn, "seek payload: "+err.Error())
		return
	}

	stopPing := startStreamPing(conn)
	defer stopPing()

	written := offset
	lastAck := offset
	lastSync := offset
	committed := false

	for {
		_ = conn.SetReadDeadline(time.Now().Add(streamReadIdle))
		mt, data, err := conn.ReadMessage()
		if err != nil {
			_ = f.Sync()
			return
		}
		switch mt {
		case websocket.BinaryMessage:
			if written+int64(len(data)) > open.Size {
				writeStreamErr(conn, "payload exceeds declared size")
				return
			}
			n, werr := f.Write(data)
			if werr != nil {
				writeStreamErr(conn, "write payload: "+werr.Error())
				return
			}
			written += int64(n)
			if written-lastSync >= streamFsyncEvery {
				_ = f.Sync()
				lastSync = written
			}
			if written-lastAck >= streamAckEvery || written == open.Size {
				if err := writeStreamJSON(conn, streamAckMsg{Type: "ack", Offset: written}); err != nil {
					return
				}
				lastAck = written
			}
		case websocket.TextMessage:
			var msg struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				writeStreamErr(conn, "invalid text message")
				return
			}
			switch msg.Type {
			case "commit":
				if written != open.Size {
					writeStreamErr(conn, fmt.Sprintf("commit at offset %d, want %d", written, open.Size))
					return
				}
				if err := f.Sync(); err != nil {
					writeStreamErr(conn, "fsync: "+err.Error())
					return
				}
				if err := f.Close(); err != nil {
					writeStreamErr(conn, "close payload: "+err.Error())
					return
				}
				payloadClosed = true
				got, herr := hashFile(payloadPath)
				if herr != nil {
					writeStreamErr(conn, "hash payload: "+herr.Error())
					return
				}
				if got != open.Hash {
					writeStreamErr(conn, "hash mismatch (remote cache is for a different payload)")
					return
				}
				finalSize, dest, ierr := installStreamPayload(dir, payloadPath, destPath, open)
				if ierr != nil {
					writeStreamErr(conn, ierr.Error())
					return
				}
				committed = true
				_ = writeStreamJSON(conn, streamDoneMsg{Type: "done", Status: "ok", Path: dest, Size: finalSize})
				return
			default:
				writeStreamErr(conn, "unknown message type "+msg.Type)
				return
			}
		}
		if committed {
			return
		}
	}
}

func prepareStreamPayload(dir, payloadPath, destPath string, open streamOpenMsg) (int64, error) {
	meta, err := loadUploadMeta(dir)
	reset := false
	if err != nil {
		reset = true
	} else if meta.DestPath != destPath || meta.TotalSize != open.Size || meta.Compressed != open.Compressed {
		reset = true
	}
	if reset {
		_ = os.Remove(payloadPath)
		if err := saveUploadMeta(dir, uploadMeta{
			DestPath:   destPath,
			TotalSize:  open.Size,
			ChmodExec:  open.ChmodExec,
			Compressed: open.Compressed,
			Stream:     true,
		}); err != nil {
			return 0, fmt.Errorf("save meta: %w", err)
		}
	} else {
		meta.ChmodExec = open.ChmodExec
		_ = saveUploadMeta(dir, meta)
	}
	f, err := os.OpenFile(payloadPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return 0, fmt.Errorf("create payload: %w", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	off := st.Size()
	if off > open.Size {
		if err := f.Truncate(0); err != nil {
			return 0, err
		}
		off = 0
	}
	return off, nil
}

func installStreamPayload(dir, payloadPath, destPath string, open streamOpenMsg) (int64, string, error) {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return 0, "", fmt.Errorf("mkdir: %w", err)
	}
	var finalSize int64
	var err error
	if open.Compressed {
		finalSize, err = gunzipFileTo(payloadPath, destPath)
	} else {
		finalSize, err = copyFileReplace(payloadPath, destPath)
	}
	if err != nil {
		return 0, "", err
	}
	if open.ChmodExec {
		if err := os.Chmod(destPath, 0755); err != nil {
			return 0, "", fmt.Errorf("chmod: %w", err)
		}
	}
	_ = removeUploadCache(dir)
	abs, absErr := filepath.Abs(destPath)
	if absErr != nil {
		abs = destPath
	}
	return finalSize, abs, nil
}

func copyFileReplace(src, dest string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	tmp := dest + ".upload.tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	n, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return 0, copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return 0, closeErr
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return 0, err
	}
	return n, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeStreamJSON(conn *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, data)
}

func writeStreamErr(conn *websocket.Conn, msg string) {
	_ = writeStreamJSON(conn, streamErrMsg{Type: "error", Message: msg})
}

func startStreamPing(conn *websocket.Conn) func() {
	stop := make(chan struct{})
	var once sync.Once
	go func() {
		t := time.NewTicker(streamPingEvery)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				_ = conn.WriteControl(websocket.PingMessage, []byte("upload"), time.Now().Add(5*time.Second))
			}
		}
	}()
	return func() { once.Do(func() { close(stop) }) }
}
