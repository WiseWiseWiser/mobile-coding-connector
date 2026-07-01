package agentcli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type execControlMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

type wsWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsWriter) writeMessage(messageType int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(messageType, data)
}

func (w *wsWriter) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.writeMessage(websocket.TextMessage, data)
}

func sendTerminalResize(writer *wsWriter) error {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return nil
	}
	return writer.writeJSON(execControlMessage{Type: "resize", Cols: cols, Rows: rows})
}

func forwardTerminalInput(writer *wsWriter) error {
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if writeErr := writer.writeMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			return err
		}
	}
}

func normalizeTerminalReadError(err error) error {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*websocket.CloseError); ok {
		switch ce.Code {
		case websocket.CloseNormalClosure, websocket.CloseGoingAway, 4000:
			return nil
		}
		return fmt.Errorf("terminal closed: %s", ce.Text)
	}
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return nil
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}
	return err
}

func terminalDialError(err error, resp *http.Response) error {
	if resp == nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	snippet := strings.TrimSpace(string(body))
	if snippet == "" {
		return fmt.Errorf("terminal connect failed: %s", resp.Status)
	}
	return fmt.Errorf("terminal connect failed: %s: %s", resp.Status, snippet)
}
