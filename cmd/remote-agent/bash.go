package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const bashHelp = `Usage: remote-agent bash [--name <name>] [cwd]

Start an interactive shell on the remote server using the same terminal
WebSocket API as the frontend terminal page.

Arguments:
  cwd                  Optional working directory on the remote server.

Options:
  --name NAME          Session name shown to the server. Defaults to "Terminal".
  -h, --help           Show this help message.

Examples:
  remote-agent bash
  remote-agent bash ~/work/repo
  remote-agent bash --name Debug /tmp
`

type terminalControlMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

type terminalServerMessage struct {
	Type      string `json:"type"`
	Message   string `json:"message,omitempty"`
	SessionID string `json:"session_id,omitempty"`
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

func (w *wsWriter) closeDelete() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	msg := websocket.FormatCloseMessage(4000, "")
	return w.conn.WriteControl(websocket.CloseMessage, msg, noDeadline())
}

func runBash(resolve func() (*client.Client, error), args []string) error {
	var name string
	args, err := flags.
		String("--name", &name).
		Help("-h,--help", bashHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("bash takes at most 1 positional argument [cwd], got %d", len(args))
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("remote-agent bash requires an interactive terminal on stdin/stdout")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	wsURL, err := terminalWebSocketURL(cli, name, firstArg(args))
	if err != nil {
		return err
	}

	header := http.Header{}
	if cli.Token != "" {
		header.Set("Authorization", "Bearer "+cli.Token)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return terminalDialError(err, resp)
	}
	defer conn.Close()

	writer := &wsWriter{conn: conn}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("enable raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	if err := sendTerminalResize(writer); err != nil {
		return err
	}

	sigWinch := make(chan os.Signal, 1)
	signal.Notify(sigWinch, syscall.SIGWINCH)
	defer signal.Stop(sigWinch)
	go func() {
		for range sigWinch {
			_ = sendTerminalResize(writer)
		}
	}()

	readerErrCh := make(chan error, 1)
	go func() {
		readerErrCh <- readTerminalOutput(conn)
	}()

	stdinErrCh := make(chan error, 1)
	go func() {
		stdinErrCh <- forwardTerminalInput(writer)
	}()

	var runErr error
	select {
	case err := <-readerErrCh:
		runErr = normalizeTerminalReadError(err)
	case err := <-stdinErrCh:
		if err != nil && err != io.EOF {
			runErr = err
		}
	}

	_ = writer.closeDelete()
	return runErr
}

func terminalWebSocketURL(cli *client.Client, name string, cwd string) (string, error) {
	base, err := url.Parse(cli.Server)
	if err != nil {
		return "", fmt.Errorf("invalid server url %q: %w", cli.Server, err)
	}
	switch base.Scheme {
	case "http":
		base.Scheme = "ws"
	case "https":
		base.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported server scheme %q", base.Scheme)
	}
	base.Path = "/api/terminal"
	q := base.Query()
	if strings.TrimSpace(name) != "" {
		q.Set("name", name)
	}
	if strings.TrimSpace(cwd) != "" {
		q.Set("cwd", cwd)
	}
	base.RawQuery = q.Encode()
	return base.String(), nil
}

func sendTerminalResize(writer *wsWriter) error {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return nil
	}
	return writer.writeJSON(terminalControlMessage{
		Type: "resize",
		Cols: cols,
		Rows: rows,
	})
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

func readTerminalOutput(conn *websocket.Conn) error {
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		switch msgType {
		case websocket.BinaryMessage:
			if _, err := os.Stdout.Write(data); err != nil {
				return err
			}
		case websocket.TextMessage:
			handled, err := handleTerminalTextMessage(data)
			if err != nil {
				return err
			}
			if !handled {
				if _, err := os.Stdout.Write(data); err != nil {
					return err
				}
			}
		}
	}
}

func handleTerminalTextMessage(data []byte) (bool, error) {
	var msg terminalServerMessage
	if err := json.Unmarshal(data, &msg); err == nil && msg.Type != "" {
		switch msg.Type {
		case "session_id":
			return true, nil
		case "error":
			if msg.Message == "" {
				return true, fmt.Errorf("remote terminal error")
			}
			return true, fmt.Errorf("%s", msg.Message)
		default:
			if msg.Message != "" {
				_, _ = os.Stdout.WriteString(msg.Message)
				return true, nil
			}
			return true, nil
		}
	}
	return false, nil
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

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func noDeadline() time.Time {
	return time.Time{}
}
