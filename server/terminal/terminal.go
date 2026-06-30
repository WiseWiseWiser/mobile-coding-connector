package terminal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/xhd2015/agent-pro/agent/exec/tool_resolve"
	"github.com/xhd2015/ai-critic/server/encrypt"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap"
)

// SessionInfo is retained for settings package compatibility.
type SessionInfo = ptywrap.SessionInfo

// TerminalSessionsResponse is retained for backward compatibility.
type TerminalSessionsResponse = ptywrap.SessionsResponse

var (
	adapterOnce sync.Once
	adapterMgr  *ptywrap.Manager
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func manager() *ptywrap.Manager {
	adapterOnce.Do(func() {
		adapterMgr = ptywrap.NewManager()
		termCfg, _ := LoadConfig()
		opts := ptywrap.SpawnOptions{
			ExtraPaths: tool_resolve.AllExtraPaths(),
		}
		if termCfg != nil {
			opts.Shell = termCfg.Shell
			opts.ShellFlags = termCfg.ShellFlags
			opts.PS1 = termCfg.PS1
			if len(termCfg.ExtraPaths) > 0 {
				opts.ExtraPaths = append(opts.ExtraPaths, termCfg.ExtraPaths...)
			}
		}
		adapterMgr.Spawn = opts
	})
	return adapterMgr
}

// RegisterAPI registers terminal routes backed by ptywrap.
func RegisterAPI(mux *http.ServeMux) {
	mgr := manager()
	ptywrap.RegisterAPIWithManager(mux, mgr)
	mux.HandleFunc("/api/terminal/config", handleConfig)
	mux.HandleFunc("/api/terminal", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ssh") == "true" {
			handleSSHWebSocket(w, r, mgr)
			return
		}
		ptywrap.HandleTerminalWebSocket(w, r, mgr)
	})
}

type sshControlMessage struct {
	Type string `json:"type"`
	Key  string `json:"key,omitempty"`
}

func handleSSHWebSocket(w http.ResponseWriter, r *http.Request, mgr *ptywrap.Manager) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	sshHost := r.URL.Query().Get("host")
	sshPortStr := r.URL.Query().Get("port")
	sshUser := r.URL.Query().Get("user")

	msgType, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}
	if msgType != websocket.TextMessage {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Expected SSH key message"}`))
		conn.Close()
		return
	}
	var msg sshControlMessage
	if err := json.Unmarshal(message, &msg); err != nil || msg.Type != "ssh_key" {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Invalid SSH key message"}`))
		conn.Close()
		return
	}

	privateKey, err := encrypt.Decrypt(msg.Key)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"error","message":"Failed to decrypt SSH key: %s"}`, err.Error())))
		conn.Close()
		return
	}
	if !strings.Contains(privateKey, "BEGIN OPENSSH PRIVATE KEY") &&
		!strings.Contains(privateKey, "BEGIN RSA PRIVATE KEY") &&
		!strings.Contains(privateKey, "BEGIN EC PRIVATE KEY") &&
		!strings.Contains(privateKey, "BEGIN DSA PRIVATE KEY") {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Invalid SSH key format. Key must be a valid private key."}`))
		conn.Close()
		return
	}

	tmpKeyFile, err := os.CreateTemp("", "ssh-key-*")
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"error","message":"Failed to create temp key file: %s"}`, err.Error())))
		conn.Close()
		return
	}
	defer os.Remove(tmpKeyFile.Name())
	if !strings.HasSuffix(privateKey, "\n") {
		privateKey += "\n"
	}
	if _, err := tmpKeyFile.WriteString(privateKey); err != nil {
		tmpKeyFile.Close()
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"error","message":"Failed to write key file: %s"}`, err.Error())))
		conn.Close()
		return
	}
	tmpKeyFile.Close()
	if err := os.Chmod(tmpKeyFile.Name(), 0600); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"error","message":"Failed to set key file permissions: %s"}`, err.Error())))
		conn.Close()
		return
	}

	sshPort := 22
	if sshPortStr != "" {
		if p, err := strconv.Atoi(sshPortStr); err == nil {
			sshPort = p
		}
	}
	sshName := fmt.Sprintf("%s@%s", sshUser, sshHost)
	sessionID, err := createSSHSession(mgr, sshName, sshHost, sshPort, sshUser, tmpKeyFile.Name())
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"error","message":"%s"}`, err.Error())))
		conn.Close()
		return
	}
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"session_id","session_id":"%s"}`, sessionID)))
	ptywrap.ServeSessionWebSocket(conn, sessionID, r.URL.Query().Get("attach_mode"), mgr)
}

func createSSHSession(mgr *ptywrap.Manager, name, host string, port int, user, sshKeyPath string) (string, error) {
	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
	}
	if port != 0 && port != 22 {
		sshArgs = append(sshArgs, "-p", fmt.Sprintf("%d", port))
	}
	if sshKeyPath != "" {
		sshArgs = append(sshArgs, "-i", sshKeyPath)
	}
	sshArgs = append(sshArgs, "-t", fmt.Sprintf("%s@%s", user, host))

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Env = tool_resolve.AppendExtraPaths(cmd.Env)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("start ssh pty: %w", err)
	}
	pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})
	return mgr.RegisterExternal(name, fmt.Sprintf("%s@%s", user, host), []string{"ssh"}, cmd, ptmx)
}