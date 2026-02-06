package terminal

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// ResizeMessage is sent from client to resize the terminal
type ResizeMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// RegisterAPI registers the terminal WebSocket endpoint
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/terminal", handleTerminalWebSocket)
}

func handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error getting current directory: "+err.Error()))
		return
	}

	// Start bash with login and interactive flags
	cmd := exec.Command("bash", "--login", "-i")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start the command with a PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error starting PTY: "+err.Error()))
		return
	}

	defer func() {
		ptmx.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Set initial terminal size
	pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	done := make(chan struct{})

	// Read from PTY and send to WebSocket
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					conn.WriteMessage(websocket.TextMessage, []byte("\r\n[Terminal closed]"))
				}
				return
			}
			if n > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	// Read from WebSocket and write to PTY
	go func() {
		for {
			msgType, message, err := conn.ReadMessage()
			if err != nil {
				ptmx.Close()
				return
			}

			// Check if it's a resize message (JSON)
			if msgType == websocket.TextMessage {
				var resize ResizeMessage
				if err := json.Unmarshal(message, &resize); err == nil && resize.Type == "resize" {
					pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(resize.Rows),
						Cols: uint16(resize.Cols),
					})
					continue
				}
			}

			// Regular input
			_, err = ptmx.Write(message)
			if err != nil {
				return
			}
		}
	}()

	// Wait for PTY to close
	<-done
}
