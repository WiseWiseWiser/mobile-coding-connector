package adaptertest

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/script/lib"
)

// Request is the doctest harness request for ai-critic adapter tests.
type Request struct {
	Phase string

	ServerPort  int
	ServerURL   string
	SessionName string
	SessionCwd  string
}

// Response is the doctest harness response.
type Response struct {
	SessionID      string
	ListJSON       map[string]interface{}
	ListStatus     int
	WSCreateOutput string
}

// Run executes an adapter regression phase.
func Run(t *testing.T, req *Request) (*Response, error) {
	switch req.Phase {
	case "ws-create":
		return runWSCreateRegression(t, req)
	case "list-shape":
		return runListShapeRegression(t, req)
	default:
		return nil, fmt.Errorf("unknown phase %q", req.Phase)
	}
}

// StartAICriticServer builds and starts ai-critic-server for adapter tests.
func StartAICriticServer(t *testing.T) (base string, port int, cleanup func()) {
	t.Helper()
	moduleRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	safe := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-ptywrap-adapter-"+safe)
	cmd := exec.Command("go", "build", "-o", serverBin, ".")
	cmd.Dir = moduleRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build server: %v\n%s", err, out)
	}
	t.Cleanup(func() { os.Remove(serverBin) })

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		t.Fatal(err)
	}

	port, err = pickFreePort(24712)
	if err != nil {
		t.Fatal(err)
	}
	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(port), "--credentials-file", credFile)
	serverCmd.Dir = configHome
	serverCmd.Env = lib.AppendTestServerEnv(os.Environ(), configHome)
	if err := serverCmd.Start(); err != nil {
		t.Fatal(err)
	}
	cleanup = func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(150 * time.Millisecond)
			serverCmd.Process.Kill()
		}
	}
	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
		cleanup()
		t.Fatal(err)
	}
	base = fmt.Sprintf("http://127.0.0.1:%d", port)
	return base, port, cleanup
}

func runWSCreateRegression(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	base := req.ServerURL
	if base == "" {
		return nil, fmt.Errorf("ServerURL not set")
	}
	name := req.SessionName
	if name == "" {
		name = "adapter-test"
	}
	cwd := req.SessionCwd
	if cwd == "" {
		cwd = t.TempDir()
	}
	q := url.Values{"name": {name}, "cwd": {cwd}}
	ws, err := wsDial(base, q.Encode())
	if err != nil {
		return nil, err
	}
	defer ws.Close()
	id, err := readSessionID(ws)
	if err != nil {
		return nil, err
	}
	resp.SessionID = id
	out, _ := collectWS(ws, 2*time.Second)
	resp.WSCreateOutput = out
	return resp, nil
}

func runListShapeRegression(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	base := req.ServerURL
	if base == "" {
		return nil, fmt.Errorf("ServerURL not set")
	}
	cwd := t.TempDir()
	q := url.Values{"name": {"list-shape-seed"}, "cwd": {cwd}}
	ws, err := wsDial(base, q.Encode())
	if err != nil {
		return nil, err
	}
	_, _ = readSessionID(ws)
	ws.Close()

	httpResp, err := http.Get(base + "/api/terminal/sessions")
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	resp.ListStatus = httpResp.StatusCode
	resp.ListJSON = parsed
	return resp, nil
}

func wsDial(base, query string) (*websocket.Conn, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	u.Scheme = "ws"
	u.Path = "/api/terminal"
	u.RawQuery = query
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	return conn, err
}

func readSessionID(conn *websocket.Conn) (string, error) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return "", err
		}
		var m struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
		}
		if json.Unmarshal(msg, &m) == nil && m.Type == "session_id" {
			return m.SessionID, nil
		}
	}
	return "", fmt.Errorf("timeout waiting for session_id")
}

func collectWS(conn *websocket.Conn, wait time.Duration) (string, error) {
	var buf strings.Builder
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if buf.Len() > 0 {
				return buf.String(), nil
			}
			continue
		}
		buf.Write(msg)
	}
	return buf.String(), nil
}

func findModuleRoot() (string, error) {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		for dir := root; ; dir = filepath.Dir(dir) {
			if isAICriticModuleRoot(dir) {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	_, file, _, ok := runtime.Caller(1)
	if ok {
		for dir := filepath.Dir(file); ; dir = filepath.Dir(dir) {
			if isAICriticModuleRoot(dir) {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if isAICriticModuleRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("ai-critic module root not found")
		}
		dir = parent
	}
}

func isAICriticModuleRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "server")); err != nil {
		return false
	}
	return true
}

func pickFreePort(base int) (int, error) {
	for port := base; port < base+200; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port near %d", base)
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}