# Remote-Agent Machine Analyse-Files Doctests

Doctests for `remote-agent machine analyse-files`: full `$HOME` scan with
per-entry streamed blocks (children, semantic enrichers, git/node_modules aggregates)
and a server-rendered summary.

Most leaves are **L2 in-process** (`machineanalyse.RegisterAPIForHome` +
`agentcli.Run`). One sparse **L3 e2e** smoke keeps the product binary path.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `machineanalyse.RegisterAPIForHome` on a local
mux + `agentcli.Run` (no product binaries). One sparse **L3 e2e** smoke keeps the
binary path (`UseCLI` + `label: heavy, e2e`): `stream/basic`.

**L2** serves analyse-files SSE in-process via `RegisterAPIForHome(mux, serverHome)`
(no process `HOME` mutation). **L3 smoke** still runs `ai-critic-server` with
`HOME=serverHome` and `remote-agent`. Analyse-files walks **every** immediate child
of server home (dirs and files), deep-walks for sizes, prints one completed entry
block at a time via SSE, then emits a summary block and structured `done` frame.

**Participants**

- **L2: agentcli.Run** — in-process `machine analyse-files` CLI against the local mux.
- **L2: machineanalyse HTTP** — `RegisterAPIForHome` on ephemeral port.
- **L3: remote-agent + ai-critic-server subprocesses** — only for `stream/basic` smoke.
- **serverHome** — temp fake machine home seeded per leaf profile.
- **agentHome** — temp dir for (L3) agent config.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/machine-analyse-files-doctest-<id>/` for L3 shared binaries (file lock +
  flock). Helpers use the variable directly, not `os.Getenv`.

**Behaviors**

- `machine analyse-files` streams `home: <path>` then per-entry blocks:
  `> <name>`, immediate children (`> child  <size>`), semantic lines (tool dirs),
  optional aggregates (`git-dirs`, `worktrees`, `node_modules N dirs`).
- Top-level files show `size` and `lines` (or `lines (binary)`).
- Summary ends with `analyse-files summary`, rollups, tool-specific lines only when
  indicator dirs exist, and largest entries.
- Entry blocks appear in alphabetical order by entry name.

## Version

0.0.3

## Decision Tree

```
[remote-agent machine analyse-files]
 |
 +-- stream/                              (GROUP)  SSE streamed HOME scan
      |
      +-- basic/                          (LEAF)   exit 0; home + summary; > headers
      +-- codex-semantic/                 (LEAF)   .codex children before semantic; summary codex
      +-- file-lines/                     (LEAF)   text lines N; binary lines (binary)
      +-- git-dirs/                       (LEAF)   git entry shows git-dirs; plain omits
      +-- node-modules/                   (LEAF)   child node_modules + recursive dir count
      +-- entry-order/                    (LEAF)   blocks sorted alphabetically
      +-- topic-absent/                   (LEAF)   no .grok → summary omits grok lines
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `stream/basic` | Exit 0; `home:` line; `analyse-files summary`; entry blocks with `>` headers |
| 2 | `stream/codex-semantic` | `.codex` block: children before semantic; sessions/skills counts; summary codex lines |
| 3 | `stream/file-lines` | Text file shows `lines N`; binary shows `lines (binary)` |
| 4 | `stream/git-dirs` | Entry with git repo shows `git-dirs 1`; entry without omits line |
| 5 | `stream/node-modules` | Entry with child `node_modules` AND `node_modules N dirs` aggregate |
| 6 | `stream/entry-order` | Blocks sorted alphabetically by entry name |
| 7 | `stream/topic-absent` | When `.grok` absent, summary omits grok lines |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Stream endpoint (default mode) | stream/* |
| Top-level dir entry | stream/basic, stream/codex-semantic, stream/git-dirs, stream/node-modules, stream/entry-order |
| Top-level file entry | stream/file-lines, stream/basic, stream/entry-order |
| `.codex` semantic enricher | stream/codex-semantic, stream/entry-order, stream/topic-absent |
| `.grok` topic-present rule | stream/topic-absent (absent); others may omit `.grok` |
| Git repo discovery | stream/git-dirs |
| `node_modules` child vs recursive count | stream/node-modules |
| Alphabetical entry ordering | stream/entry-order |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-machine-analyse-files
doctest test ./tests/remote-agent-machine-analyse-files/...          # unlabeled L2 mass
doctest test --label e2e ./tests/remote-agent-machine-analyse-files/...  # ~1 L3 smoke
go test ./server/machineanalyse/... -count=1
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
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/machineanalyse"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run.
var agentcliInProcessMu sync.Mutex

type Request struct {
	Args   []string
	Server string
	Token  string

	// UseCLI forces the L3 product-binary path. Default false → L2 in-process.
	UseCLI bool
	E2E    bool

	// SeedProfile selects the serverHome fixture set (set by leaf Setup).
	SeedProfile string
}

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseCLI || req.E2E)
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ServerHome string
	AgentHome  string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if req.SeedProfile == "" {
		req.SeedProfile = "basic"
	}
	if len(req.Args) == 0 {
		req.Args = []string{"machine", "analyse-files"}
	}

	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)

	serverHome, err := os.MkdirTemp("", "machine-analyse-server-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(serverHome) })
	resp.ServerHome = serverHome

	if err := seedAnalyseServerHome(t, serverHome, req.SeedProfile, d.DOCTEST_ROOT); err != nil {
		return nil, err
	}

	agentHome, err := os.MkdirTemp("", "machine-analyse-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, moduleRoot, cacheDir, serverHome, agentHome)
	}
	return runInProcessL2(t, d, req, resp, serverHome)
}

func runInProcessL2(t *testing.T, d *session.Doctest, req *Request, resp *Response, serverHome string) (*Response, error) {
	t.Helper()

	mux := http.NewServeMux()
	machineanalyse.RegisterAPIForHome(mux, serverHome)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen in-process analyse server: %w", err)
	}
	serverPort := ln.Addr().(*net.TCPAddr).Port
	resp.ServerPort = serverPort
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	}

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 10*time.Second); err != nil {
		return nil, err
	}
	t.Logf("L2 in-process analyse server on %s home=%s", serverURL, serverHome)

	return finishAnalyseRun(t, req, resp, serverURL, func(argv []string) (int, string, string, error) {
		return runAgentInProcess(argv)
	}, true)
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, moduleRoot, cacheDir, serverHome, agentHome string) (*Response, error) {
	t.Helper()

	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	credDir := filepath.Join(serverHome, ".ai-critic")
	if err := os.MkdirAll(credDir, 0755); err != nil {
		return nil, err
	}
	credFile := filepath.Join(credDir, "server-credentials")
	if err := os.WriteFile(credFile, []byte(req.Token+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("write credentials: %w", err)
	}

	remoteConfigPath := filepath.Join(agentHome, ".ai-critic", "remote-agent-config.json")
	if err := os.MkdirAll(filepath.Dir(remoteConfigPath), 0755); err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort := pickFreePort(portBase)
	resp.ServerPort = serverPort

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	if err := writeRemoteAgentConfig(remoteConfigPath, normalizedServer, req.Token); err != nil {
		return nil, err
	}

	killPort(serverPort)

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = serverHome
	serverCmd.Env = stripEnvPrefix(os.Environ(), "HOME=")
	serverCmd.Env = stripEnvPrefix(serverCmd.Env, lib.EnvAI_CRITIC_HOME+"=")
	serverCmd.Env = append(serverCmd.Env, "HOME="+serverHome, "AI_CRITIC_NO_OPEN_BROWSER=1")
	if err := serverCmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}
	t.Cleanup(func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(150 * time.Millisecond)
			serverCmd.Process.Kill()
		}
	})

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
		return nil, err
	}

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	return finishAnalyseRun(t, req, resp, serverURL, func(argv []string) (int, string, string, error) {
		return runAgent(agentBin, argv, agentEnv)
	}, false)
}

func finishAnalyseRun(
	t *testing.T,
	req *Request,
	resp *Response,
	serverURL string,
	runCLI func(argv []string) (int, string, string, error),
	inProcess bool,
) (*Response, error) {
	t.Helper()

	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)

	mode := "L2-inprocess"
	if !inProcess {
		mode = "L3-binary"
	}
	t.Logf("%s remote-agent argv: %v", mode, argv)

	exitCode, stdout, stderr, runErr := runCLI(argv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	return resp, nil
}

func runAgentInProcess(argv []string) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return 0, "", "", err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return 0, "", "", err
	}

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW

	runErr := agentcli.Run(agentcli.RemoteProfile(), argv)

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr

	outBytes, _ := io.ReadAll(stdoutR)
	errBytes, _ := io.ReadAll(stderrR)
	_ = stdoutR.Close()
	_ = stderrR.Close()

	stdout := string(outBytes)
	stderr := string(errBytes)
	exitCode := 0
	if runErr != nil {
		stderr += fmt.Sprintf("Error: %v\n", runErr)
		exitCode = 1
	}
	return exitCode, stdout, stderr, nil
}

func runAgent(bin string, argv, env []string) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, "", "", runErr
		}
	}
	return exitCode, outBuf.String(), errBuf.String(), nil
}

type remoteAgentConfigFile struct {
	Default string            `json:"default,omitempty"`
	Domains []domainConfigRow `json:"domains"`
}

type domainConfigRow struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

func writeRemoteAgentConfig(path, server, token string) error {
	cfg := remoteAgentConfigFile{
		Default: server,
		Domains: []domainConfigRow{{Server: server, Token: token}},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func portBaseFromTestName(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return 30000 + (hash % 1000)
}

func pickFreePort(base int) int {
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	panic(fmt.Sprintf("no free port near %d", base))
}

func killPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return
	}
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(out))) {
		_ = exec.Command("kill", "-9", pidStr).Run()
	}
}

func stripEnvPrefix(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
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
```
