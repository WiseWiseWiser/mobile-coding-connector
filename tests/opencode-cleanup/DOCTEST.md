# Opencode Serve Cleanup Helper Doctests

Package-level tests for `script/lib/opencode_cleanup.go`: discovering `opencode serve`
child PIDs from the persisted registry and via port listeners, verifying process
command before kill, and clearing registries after cleanup.

# DSN (Domain Specific Notion)

**Participants**

- **Config home** — isolated `AI_CRITIC_HOME` temp dir holding
  `opencode-serve-children.json` (and lock file when implemented).
- **Registry fixture** — JSON list of headless/custom agent children with pid,
  port, session_id, agent_id.
- **Opencode serve child** — real or fake `opencode serve --port N` subprocess
  bound to a TCP port; fake built from grok-integration `testdata/fake-opencode`.
- **Wrong-process listener** — non-opencode TCP server (stdlib `net/http`) used
  to prove kill skips unverified PIDs.
- **Cleanup helpers** — `CollectOpencodeServePIDs`, `KillOpencodeServePIDs`,
  `CleanupOpencodeServe` in `script/lib` (implementer adds).

**Behaviors**

- Collect reads registry children and returns their PIDs; also discovers listeners
  on `extraPorts` via `lsof`.
- Kill verifies each PID runs `opencode serve` (via `ps`) before SIGTERM/SIGKILL;
  rejects or skips wrong processes.
- After kill-all, registry file is empty or removed.
- No `pkill -f`; discovery uses registry + `lsof -ti tcp:PORT`.

## Version

0.0.2

## Decision Tree

```
[opencode serve cleanup helpers]
 |
 +-- collect/                              (grouping: PID discovery)
 |    |
 |    +-- registry-read-returns-children/  (LEAF) fixture JSON → Collect returns PIDs
 |    +-- port-discovery-finds-listener/   (LEAF) fake opencode on port → Collect finds PID
 |
 +-- kill/                                 (grouping: verified kill + registry clear)
      |
      +-- verify-rejects-wrong-process/    (LEAF) registry points at http.Server → not killed
      +-- kill-terminates-child/          (LEAF) fake opencode serve → port closed
      +-- kill-clears-registry/           (LEAF) CleanupOpencodeServe → registry empty
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `collect/registry-read-returns-children` | Fixture registry JSON yields matching child PIDs |
| 2 | `collect/port-discovery-finds-listener` | Extra port with fake opencode listener discovered via lsof |
| 3 | `kill/verify-rejects-wrong-process` | Kill skips PID that is not `opencode serve` |
| 4 | `kill/kill-terminates-child` | Kill terminates fake opencode serve child; port no longer listening |
| 5 | `kill/kill-clears-registry` | Cleanup clears `opencode-serve-children.json` after kill |

## How to Run

```sh
doctest vet ./tests/opencode-cleanup
doctest test ./tests/opencode-cleanup/...
```

```go
import (
	"encoding/json"
	"fmt"
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
	OpCollect = "collect"
	OpKill     = "kill"
	OpCleanup  = "cleanup"
)

type Request struct {
	Op string

	ConfigHome string

	// Fixture registry (collect + kill leaves).
	WriteFixture bool
	FixturePID   int
	FixturePort  int

	// Subprocess scenarios.
	StartFakeOpenCode bool
	StartWrongProcess bool
	UseRegistryPID    bool

	ExtraPorts []int
	KillPIDs   []int
}

type Response struct {
	ConfigHome string

	CollectedPIDs []int
	CollectErr    error

	KillErr      error
	KillSkipped  []int
	KillKilled   []int

	CleanupErr error

	RegistryRaw   string
	RegistryEmpty bool

	ListenerPID   int
	PortListening bool
	ProcessAlive  bool

	FakeOpenCodePID int
	FakeOpenCodePort int
}

func childrenRegistryPath(configHome string) string {
	return filepath.Join(configHome, "opencode-serve-children.json")
}

func writeFixtureRegistry(t *testing.T, configHome string, pid, port int) {
	t.Helper()
	reg := struct {
		Children []map[string]interface{} `json:"children"`
	}{
		Children: []map[string]interface{}{
			{
				"kind":        "headless-agent",
				"session_id":  "agent-session-fixture",
				"pid":         pid,
				"port":        port,
				"project_dir": t.TempDir(),
				"agent_id":    "grok",
				"started_at":  time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture registry: %v", err)
	}
	path := childrenRegistryPath(configHome)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir registry dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write fixture registry: %v", err)
	}
}

func readRegistryRaw(configHome string) (string, error) {
	data, err := os.ReadFile(childrenRegistryPath(configHome))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func registryIsEmpty(configHome string) bool {
	raw, err := readRegistryRaw(configHome)
	if err != nil || raw == "" {
		return true
	}
	var reg struct {
		Children []json.RawMessage `json:"children"`
	}
	if json.Unmarshal([]byte(raw), &reg) != nil {
		return false
	}
	return len(reg.Children) == 0
}

func findFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
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

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func pidOnPort(port int) (int, error) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).Output()
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return 0, fmt.Errorf("no pid on port %d", port)
	}
	return strconv.Atoi(fields[0])
}

func fakeOpenCodeSrcDir() string {
	root := DOCTEST_ROOT
	if root == "" {
		root = "."
	}
	return filepath.Join(root, "..", "..", "server", "agents", "tests", "grok-integration", "testdata", "fake-opencode")
}

func buildFakeOpenCodeBin(t *testing.T, binDir string) string {
	t.Helper()
	binPath := filepath.Join(binDir, "opencode")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = fakeOpenCodeSrcDir()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake opencode: %v\n%s", err, out)
	}
	return binPath
}

func startFakeOpenCodeServe(t *testing.T, port int) *exec.Cmd {
	t.Helper()
	binDir := filepath.Join(t.TempDir(), "fake-oc-bin")
	_ = os.MkdirAll(binDir, 0755)
	bin := buildFakeOpenCodeBin(t, binDir)
	cmd := exec.Command(bin, "serve", "--port", strconv.Itoa(port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fake opencode: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if isPortListening(port) {
			return cmd
		}
		time.Sleep(50 * time.Millisecond)
	}
	cmd.Process.Kill()
	t.Fatalf("fake opencode did not listen on port %d", port)
	return nil
}

func startWrongProcessListener(t *testing.T, port int) (*http.Server, int) {
	t.Helper()
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("listen wrong-process port: %v", err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}
	go srv.Serve(ln)
	deadline := time.Now().Add(3 * time.Second)
	var pid int
	for time.Now().Before(deadline) {
		pid, err = pidOnPort(port)
		if err == nil && pid > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if pid <= 0 {
		srv.Close()
		t.Fatalf("wrong-process listener did not bind port %d", port)
	}
	t.Cleanup(func() { srv.Close() })
	return srv, pid
}

func ensureConfigHome(t *testing.T, req *Request) string {
	if req.ConfigHome != "" {
		return req.ConfigHome
	}
	home, err := lib.CreateTestConfigHome()
	if err != nil {
		t.Fatalf("CreateTestConfigHome: %v", err)
	}
	os.Setenv(lib.EnvAI_CRITIC_HOME, home)
	t.Cleanup(func() {
		os.Unsetenv(lib.EnvAI_CRITIC_HOME)
		os.RemoveAll(home)
	})
	req.ConfigHome = home
	return home
}

func runCollect(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	home := ensureConfigHome(t, req)
	resp.ConfigHome = home

	if req.WriteFixture {
		pid := req.FixturePID
		port := req.FixturePort
		if pid <= 0 {
			pid = os.Getpid()
		}
		if port <= 0 {
			port = findFreePort(t)
		}
		writeFixtureRegistry(t, home, pid, port)
	}

	ports := req.ExtraPorts
	if req.StartFakeOpenCode {
		port := findFreePort(t)
		cmd := startFakeOpenCodeServe(t, port)
		resp.FakeOpenCodePort = port
		if cmd.Process != nil {
			resp.FakeOpenCodePID = cmd.Process.Pid
		}
		ports = append(ports, port)
		t.Cleanup(func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		})
	}

	pids, err := lib.CollectOpencodeServePIDs(home, ports...)
	resp.CollectedPIDs = pids
	resp.CollectErr = err

	raw, _ := readRegistryRaw(home)
	resp.RegistryRaw = raw
	return resp, nil
}

func runKill(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	home := ensureConfigHome(t, req)
	resp.ConfigHome = home

	port := req.FixturePort
	if port <= 0 {
		port = findFreePort(t)
	}
	resp.FakeOpenCodePort = port

	var targetPID int
	if req.StartWrongProcess {
		_, pid := startWrongProcessListener(t, port)
		targetPID = pid
		resp.ListenerPID = pid
		if req.WriteFixture || req.UseRegistryPID {
			writeFixtureRegistry(t, home, pid, port)
		}
	} else if req.StartFakeOpenCode {
		cmd := startFakeOpenCodeServe(t, port)
		if cmd.Process != nil {
			targetPID = cmd.Process.Pid
			resp.FakeOpenCodePID = targetPID
		}
		if req.WriteFixture || req.UseRegistryPID {
			writeFixtureRegistry(t, home, targetPID, port)
		}
	} else if req.WriteFixture {
		pid := req.FixturePID
		if pid <= 0 {
			pid = os.Getpid()
		}
		targetPID = pid
		writeFixtureRegistry(t, home, pid, port)
	}

	pids := req.KillPIDs
	if len(pids) == 0 && targetPID > 0 {
		pids = []int{targetPID}
	}

	skipped, killed, err := lib.KillOpencodeServePIDs(home, pids)
	resp.KillSkipped = skipped
	resp.KillKilled = killed
	resp.KillErr = err

	if port > 0 {
		resp.PortListening = isPortListening(port)
	}
	if targetPID > 0 {
		resp.ProcessAlive = isProcessAlive(targetPID)
	}

	raw, _ := readRegistryRaw(home)
	resp.RegistryRaw = raw
	resp.RegistryEmpty = registryIsEmpty(home)
	return resp, nil
}

func runCleanup(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	home := ensureConfigHome(t, req)
	resp.ConfigHome = home

	port := findFreePort(t)
	resp.FakeOpenCodePort = port
	cmd := startFakeOpenCodeServe(t, port)
	if cmd.Process != nil {
		resp.FakeOpenCodePID = cmd.Process.Pid
	}
	writeFixtureRegistry(t, home, cmd.Process.Pid, port)

	err := lib.CleanupOpencodeServe(home, port)
	resp.CleanupErr = err
	resp.PortListening = isPortListening(port)
	if cmd.Process != nil {
		resp.ProcessAlive = isProcessAlive(cmd.Process.Pid)
	}
	raw, _ := readRegistryRaw(home)
	resp.RegistryRaw = raw
	resp.RegistryEmpty = registryIsEmpty(home)
	return resp, nil
}

func Run(t *testing.T, req *Request) (*Response, error) {
	if req.Op == "" {
		return nil, fmt.Errorf("Op is required")
	}
	switch req.Op {
	case OpCollect:
		return runCollect(t, req)
	case OpKill:
		return runKill(t, req)
	case OpCleanup:
		return runCleanup(t, req)
	default:
		return nil, fmt.Errorf("unknown Op: %q", req.Op)
	}
}
```
