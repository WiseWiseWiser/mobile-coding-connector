# Server Opencode Serve Orphan Cleanup Doctests

End-to-end tests starting the `ai-critic-server` binary, launching a headless grok
agent session, and verifying no `opencode serve` orphans remain after explicit stop
or graceful server shutdown.

# DSN (Domain Specific Notion)

**Participants**

- **ai-critic-server subprocess** — built from repo root; isolated `AI_CRITIC_HOME`
  with test credentials.
- **Agent sessions API** — `POST /api/agents/sessions` launches grok;
  `DELETE /api/agents/sessions?id=` stops session.
- **Opencode serve child** — fake or real `opencode serve` on session port (fake
  preferred for speed; real when in PATH for integration confidence).
- **Children registry** — `opencode-serve-children.json` under config home.
- **Cleanup helper** — `lib.CleanupOpencodeServe` used by `stopServer` harness.

**Behaviors**

- After agent session stop + `stopServer`, zero listeners on session port.
- After SIGTERM graceful shutdown (no explicit session stop), session port closed
  and registry empty.
- Discovery via registry + `lsof`; no `pkill -f`.

## Version

0.0.2

## Decision Tree

```
[server orphan cleanup]
 |
 +-- agent-session/                           (grouping: explicit stop API)
 |    +-- no-orphan-after-agent-session-stop/  (LEAF) launch → DELETE stop → no listener
 |
 +-- shutdown/                                (grouping: graceful server exit)
      +-- no-orphan-after-server-shutdown/     (LEAF) launch → SIGTERM server → no listener
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `agent-session/no-orphan-after-agent-session-stop` | POST launch + DELETE stop leaves no session port listener |
| 2 | `shutdown/no-orphan-after-server-shutdown` | Graceful server shutdown cleans child port and registry |

## How to Run

```sh
doctest vet ./tests/server/cleanup
doctest test ./tests/server/cleanup/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

const (
	ScenarioAgentStop   = "agent-stop"
	ScenarioShutdown    = "shutdown"
)

type Request struct {
	Scenario string

	ServerPort  int
	TimeoutSecs int

	AgentID string
	UseFakeOpenCode bool
}

type Response struct {
	ServerPort int
	ConfigHome string

	SessionID   string
	SessionPort int

	RegistryRaw   string
	RegistryEmpty bool
	PortListening bool

	ServerLogs string
}

func findGoModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func pickServerPort(t *testing.T, base int) int {
	t.Helper()
	if base <= 0 {
		base = 24712
	}
	hash := 0
	for _, c := range t.Name() {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	port := base + (hash % 100)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		for p := port + 1; p < port+50; p++ {
			ln, err = net.Listen("tcp", fmt.Sprintf(":%d", p))
			if err == nil {
				port = p
				break
			}
		}
	}
	if ln != nil {
		ln.Close()
	}
	return port
}

func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func registryPath(configHome string) string {
	return filepath.Join(configHome, "opencode-serve-children.json")
}

func registryIsEmpty(configHome string) bool {
	data, err := os.ReadFile(registryPath(configHome))
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return true
	}
	var reg struct {
		Children []json.RawMessage `json:"children"`
	}
	if json.Unmarshal(data, &reg) != nil {
		return false
	}
	return len(reg.Children) == 0
}

func fakeOpenCodeSrcDir() string {
	root := DOCTEST_ROOT
	if root == "" {
		root = "."
	}
	return filepath.Join(root, "..", "..", "server", "agents", "tests", "grok-integration", "testdata", "fake-opencode")
}

func buildFakeOpenCode(t *testing.T, binDir string) {
	t.Helper()
	_ = os.MkdirAll(binDir, 0755)
	bin := filepath.Join(binDir, "opencode")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = fakeOpenCodeSrcDir()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake opencode: %v\n%s", err, out)
	}
}

func httpGetOK(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func apiRequest(method, url, body, token string) (*http.Response, []byte, error) {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, b, nil
}

func stopServerProcess(cmd *exec.Cmd, configHome string, extraPorts ...int) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Signal(os.Interrupt)
		time.Sleep(300 * time.Millisecond)
		cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(200 * time.Millisecond)
		cmd.Process.Kill()
	}
	_ = lib.CleanupOpencodeServe(configHome, extraPorts...)
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 45
	}
	if req.AgentID == "" {
		req.AgentID = "grok"
	}

	moduleRoot, err := findGoModuleRoot()
	if err != nil {
		return nil, err
	}
	// From tests/server/cleanup leaf, module root is DOCTEST_ROOT/../..
	if strings.HasSuffix(moduleRoot, "cleanup") || strings.Contains(moduleRoot, "server/cleanup") {
		moduleRoot = filepath.Join(DOCTEST_ROOT, "..", "..")
		if _, statErr := os.Stat(filepath.Join(moduleRoot, "go.mod")); statErr != nil {
			moduleRoot, err = findGoModuleRoot()
			if err != nil {
				return nil, err
			}
		}
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	binPath := filepath.Join(os.TempDir(), "ai-critic-cleanup-server-"+safeName)
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build server: %v\n%s", err, out)
	}
	t.Cleanup(func() { os.Remove(binPath) })

	fakeBinDir := filepath.Join(t.TempDir(), "fake-oc")
	if req.UseFakeOpenCode {
		buildFakeOpenCode(t, fakeBinDir)
	}

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	resp.ConfigHome = configHome
	os.Setenv(lib.EnvAI_CRITIC_HOME, configHome)
	t.Cleanup(func() {
		lib.CleanupOpencodeServe(configHome)
		os.Unsetenv(lib.EnvAI_CRITIC_HOME)
		os.RemoveAll(configHome)
	})

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	serverPort := pickServerPort(t, req.ServerPort)
	resp.ServerPort = serverPort

	args := []string{"--port", strconv.Itoa(serverPort), "--credentials-file", credFile}
	cmd := exec.Command(binPath, args...)
	cmd.Dir = configHome
	env := make([]string, 0, len(os.Environ())+2)
	binDir := filepath.Dir(binPath)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") {
			pathVal := e[5:]
			if req.UseFakeOpenCode {
				pathVal = fakeBinDir + string(os.PathListSeparator) + binDir + string(os.PathListSeparator) + pathVal
			} else {
				pathVal = binDir + string(os.PathListSeparator) + pathVal
			}
			env = append(env, "PATH="+pathVal)
		} else {
			env = append(env, e)
		}
	}
	cmd.Env = lib.AppendTestServerEnv(env, configHome)

	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	deadline := time.Now().Add(time.Duration(req.TimeoutSecs) * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		if httpGetOK(pingURL) {
			ready = true
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if !ready {
		stopServerProcess(cmd, configHome)
		return nil, fmt.Errorf("server not ready on port %d", serverPort)
	}

	projectDir, err := os.MkdirTemp("", "cleanup-proj-*")
	if err != nil {
		stopServerProcess(cmd, configHome)
		return nil, err
	}
	defer os.RemoveAll(projectDir)

	launchBody := fmt.Sprintf(`{"agent_id":%q,"project_dir":%q}`, req.AgentID, projectDir)
	sessionsURL := fmt.Sprintf("http://127.0.0.1:%d/api/agents/sessions", serverPort)
	launchResp, launchBytes, err := apiRequest(http.MethodPost, sessionsURL, launchBody, lib.TestPassword)
	if err != nil {
		stopServerProcess(cmd, configHome)
		return nil, fmt.Errorf("launch session: %w", err)
	}
	if launchResp.StatusCode != http.StatusOK {
		stopServerProcess(cmd, configHome)
		return nil, fmt.Errorf("launch status %d: %s", launchResp.StatusCode, string(launchBytes))
	}

	var sessionInfo struct {
		ID   string `json:"id"`
		Port int    `json:"port"`
	}
	if err := json.Unmarshal(launchBytes, &sessionInfo); err != nil {
		stopServerProcess(cmd, configHome, sessionInfo.Port)
		return nil, err
	}
	resp.SessionID = sessionInfo.ID
	resp.SessionPort = sessionInfo.Port

	// Wait for session port to listen
	for i := 0; i < 30; i++ {
		if isPortListening(sessionInfo.Port) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	switch req.Scenario {
	case ScenarioAgentStop:
		stopURL := fmt.Sprintf("%s?id=%s", sessionsURL, sessionInfo.ID)
		stopResp, stopBytes, err := apiRequest(http.MethodDelete, stopURL, "", lib.TestPassword)
		if err != nil {
			stopServerProcess(cmd, configHome, sessionInfo.Port)
			return nil, fmt.Errorf("stop session: %w", err)
		}
		if stopResp.StatusCode != http.StatusOK {
			stopServerProcess(cmd, configHome, sessionInfo.Port)
			return nil, fmt.Errorf("stop status %d: %s", stopResp.StatusCode, string(stopBytes))
		}
		time.Sleep(500 * time.Millisecond)
		stopServerProcess(cmd, configHome, sessionInfo.Port)

	case ScenarioShutdown:
		cmd.Process.Signal(syscall.SIGTERM)
		waitDone := make(chan error, 1)
		go func() { waitDone <- cmd.Wait() }()
		select {
		case <-waitDone:
		case <-time.After(10 * time.Second):
			cmd.Process.Kill()
			<-waitDone
		}
		time.Sleep(500 * time.Millisecond)
		_ = lib.CleanupOpencodeServe(configHome, sessionInfo.Port)

	default:
		stopServerProcess(cmd, configHome, sessionInfo.Port)
		return nil, fmt.Errorf("unknown Scenario: %q", req.Scenario)
	}

	if data, err := os.ReadFile(registryPath(configHome)); err == nil {
		resp.RegistryRaw = strings.TrimSpace(string(data))
	}
	resp.RegistryEmpty = registryIsEmpty(configHome)
	resp.PortListening = isPortListening(sessionInfo.Port)
	resp.ServerLogs = logBuf.String()

	return resp, nil
}
```
