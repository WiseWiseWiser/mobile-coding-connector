package webtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap"
)

type Request struct{ Scenario string }
type Response struct {
	Status       string
	SessionCount int
	Replay       bool
	Input        bool
	Pong         bool
	SameSession  bool
}

// Run uses a real foreground PTY command and the public REST/WebSocket API.
// Each scenario owns its manager/server; no user state or global env is touched.
func Run(t *testing.T, req *Request) (*Response, error) {
	t.Helper()
	mgr := ptywrap.NewManager()
	defer mgr.Close()
	mux := http.NewServeMux()
	ptywrap.RegisterAPIWithManager(mux, mgr)
	server := httptest.NewServer(mux)
	defer server.Close()
	wsbase := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/terminal?attach_mode=web"
	result := &Response{}
	list := func() ([]ptywrap.SessionInfo, error) {
		resp, err := http.Get(server.URL + "/api/terminal/sessions")
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var data ptywrap.SessionsResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		return data.Sessions, err
	}
	if req.Scenario == "missing" {
		conn, _, err := websocket.DefaultDialer.Dial(wsbase+"&session_id=not-there", nil)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		msg, err := readState(conn, "missing", nil)
		if err != nil {
			return nil, err
		}
		result.Status = msg
		sessions, err := list()
		result.SessionCount = len(sessions)
		return result, err
	}
	info, err := mgr.CreateCommand("Web terminal", t.TempDir(), []string{"/bin/sh", "-c", `printf 'WEB_READY:%s\n' "$$"; while IFS= read -r line; do if [ "$line" = '__EXIT__' ]; then exit 0; fi; printf 'OUT:%s\n' "$line"; done`})
	if err != nil {
		return nil, err
	}
	url := wsbase + "&session_id=" + info.ID
	if req.Scenario == "legacy" {
		url = strings.Replace(url, "attach_mode=web", "attach_mode=open", 1)
	}
	first, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	defer first.Close()
	var output strings.Builder
	if req.Scenario != "legacy" {
		if _, err := readState(first, "running", &output); err != nil {
			return nil, err
		}
	}
	if !strings.Contains(output.String(), "WEB_READY:") {
		if err := readUntilOutput(first, "WEB_READY:", &output); err != nil {
			return nil, err
		}
	}
	if req.Scenario == "legacy" {
		first.Close()
		sessions, err := awaitSessions(list, func(s []ptywrap.SessionInfo) bool { return len(s) == 1 && s[0].Status == "exited" })
		if err != nil {
			return nil, err
		}
		result.Status = sessions[0].Status
		return result, nil
	}
	if req.Scenario == "exit" {
		if err := first.WriteMessage(websocket.BinaryMessage, []byte("__EXIT__\n")); err != nil {
			return nil, err
		}
		status, err := readState(first, "exited", &output)
		if err != nil {
			return nil, err
		}
		result.Status = status
		again, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			return nil, err
		}
		defer again.Close()
		_, err = readState(again, "exited", nil)
		return result, err
	}
	switch req.Scenario {
	case "abrupt":
		first.UnderlyingConn().Close()
	case "normal", "delete":
		_ = first.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1000, ""), time.Now().Add(time.Second))
		first.Close()
	case "overlap": // Old socket is still alive when the browser reconnects.
	default:
		return nil, fmt.Errorf("unknown scenario %q", req.Scenario)
	}
	if req.Scenario != "overlap" {
		if _, err := awaitSessions(list, func(s []ptywrap.SessionInfo) bool { return len(s) == 1 && !s[0].Connected }); err != nil {
			return nil, err
		}
	}
	second, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	defer second.Close()
	var replay strings.Builder
	if _, err := readState(second, "running", &replay); err != nil {
		return nil, err
	}
	result.Replay = strings.Contains(replay.String(), "WEB_READY:")
	if req.Scenario == "overlap" {
		first.Close()
	}
	if err := second.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`)); err != nil {
		return nil, err
	}
	if err := readControl(second, "pong"); err != nil {
		return nil, err
	}
	result.Pong = true
	if err := second.WriteMessage(websocket.BinaryMessage, []byte("WEB_INPUT_MARKER\n")); err != nil {
		return nil, err
	}
	if err := readUntilOutput(second, "OUT:WEB_INPUT_MARKER", &replay); err != nil {
		return nil, err
	}
	result.Input = true
	sessions, err := list()
	if err != nil {
		return nil, err
	}
	result.SessionCount = len(sessions)
	result.SameSession = len(sessions) == 1 && sessions[0].ID == info.ID
	result.Status = sessions[0].Status
	if req.Scenario == "delete" {
		request, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/terminal/sessions?id="+info.ID, nil)
		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		sessions, err = list()
		if err != nil {
			return nil, err
		}
		result.SessionCount = len(sessions)
	}
	return result, nil
}

func awaitSessions(list func() ([]ptywrap.SessionInfo, error), done func([]ptywrap.SessionInfo) bool) ([]ptywrap.SessionInfo, error) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		sessions, err := list()
		if err != nil {
			return nil, err
		}
		if done(sessions) {
			return sessions, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, fmt.Errorf("session lifecycle timed out")
}

func readState(conn *websocket.Conn, want string, output *strings.Builder) (string, error) {
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			return "", err
		}
		if kind == websocket.TextMessage {
			var msg struct{ Type, Status string }
			if json.Unmarshal(data, &msg) == nil && msg.Type == "session_state" {
				if msg.Status != want {
					return "", fmt.Errorf("state = %q, want %q", msg.Status, want)
				}
				return msg.Status, nil
			}
		}
		if output != nil {
			output.Write(data)
		}
	}
}

func readUntilOutput(conn *websocket.Conn, marker string, output *strings.Builder) error {
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for !strings.Contains(output.String(), marker) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		output.Write(data)
	}
	return nil
}

func readControl(conn *websocket.Conn, want string) error {
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msg struct{ Type string }
		if json.Unmarshal(data, &msg) == nil && msg.Type == want {
			return nil
		}
	}
}
