package client

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsInitFrame  = 64 * 1024
	wsMaxFrame   = 1 * 1024 * 1024
	wsInitWindow = 512 * 1024
	wsMaxWindow  = 8 * 1024 * 1024
	wsMaxTries   = 8
)

var errUploadWSUnsupported = errors.New("server does not support websocket upload")

func isUploadWSUnsupported(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errUploadWSUnsupported) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "bad handshake")
}

type wsOpenMsg struct {
	Type       string `json:"type"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
	Compressed bool   `json:"compressed"`
	ChmodExec  bool   `json:"chmod_exec"`
}

func (c *Client) uploadFileWS(localFile, remotePath string, origSize int64, opts UploadOptions, onProgress func(UploadProgress)) (*UploadResult, error) {
	wirePath, hash, wireSize, compressed, cleanup, err := prepareUploadPayloadFile(localFile, opts.NoCompress)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upload payload: %w", err)
	}
	defer cleanup()

	if onProgress != nil {
		onProgress(UploadProgress{
			Phase:       UploadStreamStart,
			OrigBytes:   origSize,
			TotalBytes:  wireSize,
			BytesPerSec: 0,
		})
	}

	var lastOffset int64
	var lastErr error
	for try := 1; try <= wsMaxTries; try++ {
		if try > 1 && onProgress != nil {
			onProgress(UploadProgress{
				Phase:          UploadStreamResuming,
				CompletedBytes: lastOffset,
				TotalBytes:     wireSize,
				OrigBytes:      origSize,
				Attempt:        try,
				MaxAttempts:    wsMaxTries,
				Err:            lastErr,
			})
		}
		result, offset, err := c.uploadFileWSOnce(wirePath, remotePath, hash, wireSize, origSize, compressed, opts, lastOffset, onProgress)
		if err == nil {
			return result, nil
		}
		if isUploadWSUnsupported(err) {
			return nil, err
		}
		lastErr = err
		if offset > lastOffset {
			lastOffset = offset
		}
		if try == wsMaxTries {
			return nil, fmt.Errorf("upload failed at offset %d: %w", lastOffset, err)
		}
		time.Sleep(backoff(try))
	}
	return nil, fmt.Errorf("upload failed at offset %d: %w", lastOffset, lastErr)
}

func backoff(try int) time.Duration {
	d := 200 * time.Millisecond
	for i := 1; i < try && i < 4; i++ {
		d *= 2
	}
	if d > 2*time.Second {
		d = 2 * time.Second
	}
	return d
}

func (c *Client) uploadFileWSOnce(wirePath, remotePath, hash string, wireSize, origSize int64, compressed bool, opts UploadOptions, hintOffset int64, onProgress func(UploadProgress)) (*UploadResult, int64, error) {
	conn, err := c.dialUploadWS()
	if err != nil {
		return nil, hintOffset, err
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	open := wsOpenMsg{
		Type:       "open",
		Path:       remotePath,
		Size:       wireSize,
		Hash:       hash,
		Compressed: compressed,
		ChmodExec:  opts.ChmodExec,
	}
	if err := writeWSJSON(conn, open); err != nil {
		return nil, hintOffset, err
	}

	var ready struct {
		Type    string `json:"type"`
		Offset  int64  `json:"offset"`
		Message string `json:"message"`
		Path    string `json:"path"`
		Size    int64  `json:"size"`
		Status  string `json:"status"`
	}
	if err := readWSJSON(conn, &ready); err != nil {
		return nil, hintOffset, err
	}
	if ready.Type == "error" {
		return nil, hintOffset, fmt.Errorf("%s", ready.Message)
	}
	if ready.Type != "ready" {
		return nil, hintOffset, fmt.Errorf("expected ready, got %q", ready.Type)
	}
	offset := ready.Offset
	if offset < 0 || offset > wireSize {
		return nil, hintOffset, fmt.Errorf("invalid resume offset %d", offset)
	}

	f, err := os.Open(wirePath)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, err
	}

	ackCh := make(chan int64, 16)
	errCh := make(chan error, 1)
	doneCh := make(chan UploadResult, 1)
	var readOnce sync.Once
	go func() {
		for {
			mt, data, rerr := conn.ReadMessage()
			if rerr != nil {
				readOnce.Do(func() { errCh <- rerr })
				return
			}
			if mt != websocket.TextMessage {
				continue
			}
			var msg struct {
				Type    string `json:"type"`
				Offset  int64  `json:"offset"`
				Message string `json:"message"`
				Path    string `json:"path"`
				Size    int64  `json:"size"`
				Status  string `json:"status"`
			}
			if uerr := json.Unmarshal(data, &msg); uerr != nil {
				readOnce.Do(func() { errCh <- uerr })
				return
			}
			switch msg.Type {
			case "ack":
				select {
				case ackCh <- msg.Offset:
				default:
					select {
					case <-ackCh:
					default:
					}
					ackCh <- msg.Offset
				}
			case "done":
				doneCh <- UploadResult{Status: msg.Status, Path: msg.Path, Size: msg.Size}
				return
			case "error":
				readOnce.Do(func() { errCh <- fmt.Errorf("%s", msg.Message) })
				return
			}
		}
	}()

	sent := offset
	acked := offset
	frame := wsInitFrame
	window := wsInitWindow
	okFrames := 0
	t0 := time.Now()
	lastReport := time.Time{}

	buf := make([]byte, wsMaxFrame)
	for sent < wireSize {
		select {
		case err := <-errCh:
			return nil, acked, err
		case a := <-ackCh:
			if a > acked {
				acked = a
			}
			okFrames++
			if okFrames%8 == 0 && frame < wsMaxFrame {
				frame *= 2
				if frame > wsMaxFrame {
					frame = wsMaxFrame
				}
			}
			if window < wsMaxWindow {
				window *= 2
				if window > wsMaxWindow {
					window = wsMaxWindow
				}
			}
		default:
		}
		for sent-acked >= int64(window) {
			select {
			case err := <-errCh:
				return nil, acked, err
			case a := <-ackCh:
				if a > acked {
					acked = a
				}
			case <-time.After(2 * time.Second):
				if frame > wsInitFrame {
					frame /= 2
				}
				if window > wsInitWindow {
					window /= 2
				}
			}
		}
		n := frame
		remain := wireSize - sent
		if int64(n) > remain {
			n = int(remain)
		}
		got, rerr := io.ReadFull(f, buf[:n])
		if rerr != nil && !errors.Is(rerr, io.ErrUnexpectedEOF) && !errors.Is(rerr, io.EOF) {
			return nil, acked, rerr
		}
		if got == 0 {
			break
		}
		_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
		if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:got]); werr != nil {
			return nil, acked, werr
		}
		sent += int64(got)
		if onProgress != nil {
			now := time.Now()
			if lastReport.IsZero() || now.Sub(lastReport) >= 250*time.Millisecond || sent == wireSize {
				elapsed := now.Sub(t0).Seconds()
				bps := int64(0)
				if elapsed > 0 && sent > offset {
					bps = int64(float64(sent-offset) / elapsed)
				}
				onProgress(UploadProgress{
					Phase:          UploadStreamProgress,
					CompletedBytes: sent,
					TotalBytes:     wireSize,
					OrigBytes:      origSize,
					BytesPerSec:    bps,
				})
				lastReport = now
			}
		}
	}
	if sent < wireSize {
		return nil, acked, fmt.Errorf("short read: sent %d want %d", sent, wireSize)
	}

	if err := writeWSJSON(conn, map[string]string{"type": "commit"}); err != nil {
		return nil, acked, err
	}

	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()
	for {
		select {
		case res := <-doneCh:
			if res.Size == 0 {
				res.Size = origSize
			}
			if onProgress != nil {
				onProgress(UploadProgress{
					Phase:          UploadStreamProgress,
					CompletedBytes: wireSize,
					TotalBytes:     wireSize,
					OrigBytes:      origSize,
				})
			}
			return &res, wireSize, nil
		case err := <-errCh:
			return nil, acked, err
		case a := <-ackCh:
			if a > acked {
				acked = a
			}
		case <-timer.C:
			return nil, acked, fmt.Errorf("timeout waiting for commit")
		}
	}
}

func (c *Client) dialUploadWS() (*websocket.Conn, error) {
	wsURL, err := c.uploadWSURL()
	if err != nil {
		return nil, err
	}
	hdr := http.Header{}
	if c.Token != "" {
		hdr.Set("Authorization", "Bearer "+c.Token)
	}
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second
	conn, resp, err := dialer.Dial(wsURL, hdr)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusBadRequest, http.StatusNotImplemented:
				return nil, errUploadWSUnsupported
			}
			if resp.StatusCode >= 400 {
				return nil, readAPIError(resp)
			}
		}
		if strings.Contains(err.Error(), "bad handshake") {
			return nil, errUploadWSUnsupported
		}
		return nil, fmt.Errorf("upload websocket dial: %w", err)
	}
	return conn, nil
}

func (c *Client) uploadWSURL() (string, error) {
	base := c.Server
	if base == "" {
		return "", fmt.Errorf("server URL is empty")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported server scheme %q", u.Scheme)
	}
	if u.Hostname() == "localhost" {
		u.Host = strings.Replace(u.Host, "localhost", "127.0.0.1", 1)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/files/upload/ws"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func writeWSJSON(conn *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, data)
}

func readWSJSON(conn *websocket.Conn, v any) error {
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	mt, data, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	if mt != websocket.TextMessage {
		return fmt.Errorf("expected text message")
	}
	return json.Unmarshal(data, v)
}

// prepareUploadPayloadFile gzip-streams (when smaller) to a seekable file and
// returns the sha256 of the wire bytes.
func prepareUploadPayloadFile(localPath string, noCompress bool) (path string, hash string, wireSize int64, compressed bool, cleanup func(), err error) {
	nop := func() {}
	st, err := os.Stat(localPath)
	if err != nil {
		return "", "", 0, false, nop, err
	}
	orig := st.Size()
	if noCompress {
		h, herr := hashPath(localPath)
		if herr != nil {
			return "", "", 0, false, nop, herr
		}
		return localPath, h, orig, false, nop, nil
	}

	src, err := os.Open(localPath)
	if err != nil {
		return "", "", 0, false, nop, err
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "remote-agent-upload-*.gz")
	if err != nil {
		return "", "", 0, false, nop, err
	}
	tmpName := tmp.Name()
	cleanupTmp := func() { _ = os.Remove(tmpName) }

	hasher := sha256.New()
	zw, err := gzip.NewWriterLevel(io.MultiWriter(tmp, hasher), gzip.DefaultCompression)
	if err != nil {
		tmp.Close()
		cleanupTmp()
		return "", "", 0, false, nop, err
	}
	zw.Header.ModTime = time.Time{}
	if _, err := io.Copy(zw, src); err != nil {
		_ = zw.Close()
		tmp.Close()
		cleanupTmp()
		return "", "", 0, false, nop, err
	}
	if err := zw.Close(); err != nil {
		tmp.Close()
		cleanupTmp()
		return "", "", 0, false, nop, err
	}
	st2, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		cleanupTmp()
		return "", "", 0, false, nop, err
	}
	if err := tmp.Close(); err != nil {
		cleanupTmp()
		return "", "", 0, false, nop, err
	}
	if st2.Size() >= orig {
		cleanupTmp()
		h, herr := hashPath(localPath)
		if herr != nil {
			return "", "", 0, false, nop, herr
		}
		return localPath, h, orig, false, nop, nil
	}
	return tmpName, hex.EncodeToString(hasher.Sum(nil)), st2.Size(), true, cleanupTmp, nil
}

func hashPath(path string) (string, error) {
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
