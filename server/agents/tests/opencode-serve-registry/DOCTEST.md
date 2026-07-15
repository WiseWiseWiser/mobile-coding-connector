# Opencode Serve Children Registry Doctests

Package-level tests for persisted `opencode-serve-children.json` lifecycle: register on
headless/custom agent launch, remove on stop or child exit, and `CleanupAll` on shutdown.

# DSN (Domain Specific Notion)

**Participants**

- **Session manager** — `sessionMgr.launch` / `stop` for headless agents (grok, opencode).
- **Custom agent launcher** — `LaunchCustomAgent` spawns `opencode serve` with
  `OPENCODE_CONFIG_DIR` for custom agent config.
- **Children registry** — `{DataDir}/opencode-serve-children.json` with flock lock;
  kinds `headless-agent` and `custom-agent`.
- **Process monitor** — goroutine on `cmd.Wait()` removes registry entry when child exits.
- **CleanupAll** — kills all registered children + clears file (implementer adds).

**Behaviors**

- Successful `cmd.Start()` for Path A/B writes registry entry immediately (pid, port, session_id).
- `stop()` / `StopCustomAgentSession()` kills verified child and removes entry.
- External child death triggers registry removal via monitor.
- `CleanupAllOpencodeServe()` kills remaining children without prior stop.

## Version

0.0.2

## Decision Tree

```
[opencode-serve-children registry]
 |
 +-- headless-agent/                         (grouping: sessionMgr Path A)
 |    |
 |    +-- launch-writes-registry/            (LEAF) launch grok → JSON entry pid/port
 |    +-- stop-removes-registry/             (LEAF) launch + stop → entry gone, port closed
 |    +-- process-exit-removes-registry/     (LEAF) launch + kill child → entry removed
 |
 +-- custom-agent/                           (grouping: LaunchCustomAgent Path B)
 |    +-- custom-agent-launch-writes-registry/ (LEAF) custom launch → kind=custom-agent
 |
 +-- cleanup-all/                            (grouping: shutdown hook)
      +-- cleanup-all-kills-remaining/       (LEAF) launch without stop → CleanupAll clears all
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `headless-agent/launch-writes-registry` | Launch writes registry with matching pid/port |
| 2 | `headless-agent/stop-removes-registry` | Stop removes entry and closes port |
| 3 | `headless-agent/process-exit-removes-registry` | External SIGKILL removes registry entry |
| 4 | `custom-agent/custom-agent-launch-writes-registry` | Custom agent launch registers kind=custom-agent |
| 5 | `cleanup-all/cleanup-all-kills-remaining` | CleanupAll kills child and clears registry |

## Implementer exports (design notes)

Add to `server/agents/doctest_export.go`:

- `TestExported_ReadOpencodeServeChildrenRegistry() ([]OpencodeServeChildEntry, error)`
- `TestExported_CleanupAllOpencodeServe() error`
- `TestExported_LaunchCustomAgent(agentID, projectDir string) (LaunchCustomAgentResult, error)`
- `TestExported_StopCustomAgentSession(sessionID string)`
- `TestExported_WaitForRegistryRemoval(sessionID string, timeout time.Duration) error` (optional)

## How to Run

```sh
doctest vet ./server/agents/tests/opencode-serve-registry
doctest test ./server/agents/tests/opencode-serve-registry/...
```

```go
import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/agents"
)

const (
	OpLaunchRegistry   = "launch-registry"
	OpStopRegistry     = "stop-registry"
	OpExitRegistry     = "exit-registry"
	OpCustomRegistry   = "custom-registry"
	OpCleanupAll       = "cleanup-all"
)

type Request struct {
	Op string

	AgentID    string
	ProjectDir string

	UseFakeOpenCode bool
	UseRealOpenCode bool

	CustomAgentID string

	KillChildExternally bool
	CallCleanupAll      bool
	SkipStop            bool
}

type RegistryChild struct {
	Kind       string `json:"kind"`
	SessionID  string `json:"session_id"`
	PID        int    `json:"pid"`
	Port       int    `json:"port"`
	ProjectDir string `json:"project_dir"`
	AgentID    string `json:"agent_id"`
	StartedAt  string `json:"started_at"`
}

type Response struct {
	ConfigHome string

	LaunchSession *agents.AgentSessionInfo
	LaunchErr     error
	CustomLaunch  *agents.LaunchCustomAgentResult
	CustomErr     error

	RegistryChildren []RegistryChild
	RegistryRaw      string
	RegistryErr      error

	SessionPort     int
	PortListening   bool
	RegistryHasEntry bool
	RegistryEmpty   bool

	CleanupErr error
}

func childrenRegistryPath(configHome string) string {
	return filepath.Join(configHome, "opencode-serve-children.json")
}

func ensureAgentsConfigHome(t *testing.T) string {
	t.Helper()
	home, err := lib.CreateTestConfigHome()
	if err != nil {
		t.Fatalf("CreateTestConfigHome: %v", err)
	}
	os.Setenv(lib.EnvAI_CRITIC_HOME, home)
	t.Cleanup(func() {
		os.Unsetenv(lib.EnvAI_CRITIC_HOME)
		lib.CleanupOpencodeServe(home)
		os.RemoveAll(home)
	})
	return home
}

func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func fakeOpenCodeSrcDir() string {
	root := DOCTEST_ROOT
	if root == "" {
		root = "."
	}
	return filepath.Join(root, "..", "grok-integration", "testdata", "fake-opencode")
}

func installFakeOpenCode(t *testing.T) string {
	t.Helper()
	binDir := filepath.Join(t.TempDir(), "fake-oc-bin")
	_ = os.MkdirAll(binDir, 0755)
	binPath := filepath.Join(binDir, "opencode")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = fakeOpenCodeSrcDir()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake opencode: %v\n%s", err, out)
	}
	orig := os.Getenv("PATH")
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+orig)
	t.Cleanup(func() { os.Setenv("PATH", orig) })
	return binDir
}

func realOpenCodeOnPath() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

func prepOpenCodeBinary(t *testing.T, req *Request) {
	t.Helper()
	agents.TestExported_StripOpencodeResolutionForDoctest(t)
	if req.UseFakeOpenCode || (!req.UseRealOpenCode && !realOpenCodeOnPath()) {
		installFakeOpenCode(t)
		return
	}
	if req.UseRealOpenCode && !realOpenCodeOnPath() {
		t.Skip("real opencode not in PATH")
	}
}

func readRegistry(t *testing.T) ([]RegistryChild, string, error) {
	t.Helper()
	children, err := agents.TestExported_ReadOpencodeServeChildrenRegistry()
	if err != nil {
		return nil, "", err
	}
	out := make([]RegistryChild, 0, len(children))
	for _, c := range children {
		out = append(out, RegistryChild{
			Kind:       c.Kind,
			SessionID:  c.SessionID,
			PID:        c.PID,
			Port:       c.Port,
			ProjectDir: c.ProjectDir,
			AgentID:    c.AgentID,
			StartedAt:  c.StartedAt,
		})
	}
	raw := ""
	if home := os.Getenv(lib.EnvAI_CRITIC_HOME); home != "" {
		if data, readErr := os.ReadFile(childrenRegistryPath(home)); readErr == nil {
			raw = strings.TrimSpace(string(data))
		}
	}
	return out, raw, nil
}

func ensureProjectDir(t *testing.T, dir string) string {
	if dir != "" {
		return dir
	}
	d, err := os.MkdirTemp("", "registry-proj-*")
	if err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

func writeCustomAgentFixture(t *testing.T, agentID string) {
	t.Helper()
	home := t.TempDir()
	os.Setenv("HOME", home)
	t.Cleanup(func() { os.Unsetenv("HOME") })
	agentDir := filepath.Join(home, ".ai-critic", "agents", agentID)
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("mkdir custom agent: %v", err)
	}
	cfg := `{"name":"Doctest Cleanup Agent","description":"registry test","mode":"primary","tools":{}}`
	if err := os.WriteFile(filepath.Join(agentDir, "agent.json"), []byte(cfg), 0644); err != nil {
		t.Fatalf("write agent.json: %v", err)
	}
}

func runLaunchRegistry(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	resp.ConfigHome = ensureAgentsConfigHome(t)
	prepOpenCodeBinary(t, req)

	projectDir := ensureProjectDir(t, req.ProjectDir)
	agentID := req.AgentID
	if agentID == "" {
		agentID = "grok"
	}

	info, err := agents.TestExported_LaunchAgentSession(agentID, projectDir, "")
	resp.LaunchErr = err
	if err == nil {
		resp.LaunchSession = &info
		resp.SessionPort = info.Port
		t.Cleanup(func() {
			lib.CleanupOpencodeServe(resp.ConfigHome, info.Port)
			agents.TestExported_StopAgentSession(info.ID)
		})
	}

	children, raw, regErr := readRegistry(t)
	resp.RegistryChildren = children
	resp.RegistryRaw = raw
	resp.RegistryErr = regErr
	if resp.SessionPort > 0 {
		resp.PortListening = isPortListening(resp.SessionPort)
	}
	resp.RegistryHasEntry = len(children) > 0
	return resp, nil
}

func runStopRegistry(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	resp.ConfigHome = ensureAgentsConfigHome(t)
	prepOpenCodeBinary(t, req)

	projectDir := ensureProjectDir(t, req.ProjectDir)
	info, err := agents.TestExported_LaunchAgentSession("grok", projectDir, "")
	if err != nil {
		resp.LaunchErr = err
		return resp, nil
	}
	resp.LaunchSession = &info
	resp.SessionPort = info.Port
	time.Sleep(200 * time.Millisecond)

	agents.TestExported_StopAgentSession(info.ID)
	time.Sleep(300 * time.Millisecond)

	children, raw, regErr := readRegistry(t)
	resp.RegistryChildren = children
	resp.RegistryRaw = raw
	resp.RegistryErr = regErr
	resp.PortListening = isPortListening(info.Port)
	resp.RegistryEmpty = len(children) == 0
	return resp, nil
}

func runExitRegistry(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	resp.ConfigHome = ensureAgentsConfigHome(t)
	prepOpenCodeBinary(t, req)

	projectDir := ensureProjectDir(t, req.ProjectDir)
	info, err := agents.TestExported_LaunchAgentSession("grok", projectDir, "")
	if err != nil {
		resp.LaunchErr = err
		return resp, nil
	}
	resp.LaunchSession = &info
	resp.SessionPort = info.Port

	children, _, regErr := readRegistry(t)
	if regErr != nil {
		resp.RegistryErr = regErr
		return resp, nil
	}
	if len(children) == 0 {
		return resp, fmt.Errorf("no registry entry after launch")
	}
	pid := children[0].PID
	if pid > 0 {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		children, raw, err := readRegistry(t)
		if err == nil && len(children) == 0 {
			resp.RegistryChildren = children
			resp.RegistryRaw = raw
			resp.RegistryEmpty = true
			resp.PortListening = isPortListening(info.Port)
			return resp, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	children, raw, regErr := readRegistry(t)
	resp.RegistryChildren = children
	resp.RegistryRaw = raw
	resp.RegistryErr = regErr
	resp.RegistryEmpty = len(children) == 0
	resp.PortListening = isPortListening(info.Port)
	return resp, nil
}

func runCustomRegistry(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	resp.ConfigHome = ensureAgentsConfigHome(t)
	prepOpenCodeBinary(t, req)

	agentID := req.CustomAgentID
	if agentID == "" {
		agentID = "doctest-cleanup-agent"
	}
	writeCustomAgentFixture(t, agentID)
	projectDir := ensureProjectDir(t, req.ProjectDir)

	result, err := agents.TestExported_LaunchCustomAgent(agentID, projectDir)
	resp.CustomErr = err
	if err == nil {
		resp.CustomLaunch = &result
		resp.SessionPort = result.Port
		t.Cleanup(func() {
			lib.CleanupOpencodeServe(resp.ConfigHome, result.Port)
			agents.TestExported_StopCustomAgentSession(result.SessionID)
		})
	}

	children, raw, regErr := readRegistry(t)
	resp.RegistryChildren = children
	resp.RegistryRaw = raw
	resp.RegistryErr = regErr
	if resp.SessionPort > 0 {
		resp.PortListening = isPortListening(resp.SessionPort)
	}
	return resp, nil
}

func runCleanupAll(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	resp.ConfigHome = ensureAgentsConfigHome(t)
	prepOpenCodeBinary(t, req)

	projectDir := ensureProjectDir(t, req.ProjectDir)
	info, err := agents.TestExported_LaunchAgentSession("grok", projectDir, "")
	if err != nil {
		resp.LaunchErr = err
		return resp, nil
	}
	resp.LaunchSession = &info
	resp.SessionPort = info.Port
	time.Sleep(200 * time.Millisecond)

	resp.CleanupErr = agents.TestExported_CleanupAllOpencodeServe()
	time.Sleep(300 * time.Millisecond)

	children, raw, regErr := readRegistry(t)
	resp.RegistryChildren = children
	resp.RegistryRaw = raw
	resp.RegistryErr = regErr
	resp.PortListening = isPortListening(info.Port)
	resp.RegistryEmpty = len(children) == 0
	return resp, nil
}

func Run(t *testing.T, req *Request) (*Response, error) {
	if req.Op == "" {
		return nil, fmt.Errorf("Op is required")
	}
	switch req.Op {
	case OpLaunchRegistry:
		return runLaunchRegistry(t, req)
	case OpStopRegistry:
		return runStopRegistry(t, req)
	case OpExitRegistry:
		return runExitRegistry(t, req)
	case OpCustomRegistry:
		return runCustomRegistry(t, req)
	case OpCleanupAll:
		return runCleanupAll(t, req)
	default:
		return nil, fmt.Errorf("unknown Op: %q", req.Op)
	}
}
```
