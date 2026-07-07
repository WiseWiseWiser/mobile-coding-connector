# Remote-Agent Machine Analyse-Files Doctests

End-to-end tests for `remote-agent machine analyse-files`: full `$HOME` scan with
per-entry streamed blocks (children, semantic enrichers, git/node_modules aggregates)
and a server-rendered summary.

# DSN (Domain Specific Notion)

The harness models a remote machine as an isolated `serverHome` directory. The
`ai-critic-server` subprocess runs with `HOME=serverHome` and working directory
`serverHome`, so server `~` and scan scope align. The CLI runs in a separate
`agentHome` with only `remote-agent-config.json`. Analyse-files walks **every**
immediate child of server home (dirs and files), deep-walks for sizes, prints one
completed entry block at a time via SSE, then emits a summary block and structured
`done` frame.

**Participants**

- **remote-agent subprocess** — `./cmd/remote-agent`; subcommand `machine analyse-files`
  with `--server` / `--token`.
- **ai-critic-server subprocess** — ephemeral port;
  `POST /api/remote-agent/machine/analyse-files/stream` (SSE progress stream).
- **serverHome** — temp fake machine home seeded per leaf profile (`.codex` sessions,
  git repos, `node_modules`, text/binary top-level files, etc.).
- **agentHome** — temp `HOME` for `~/.ai-critic/remote-agent-config.json` only.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/machine-analyse-files-doctest-<id>/` for shared binaries (file lock +
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

0.0.2

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
doctest test -v ./tests/remote-agent-machine-analyse-files/...
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
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

type Request struct {
	Args   []string
	Server string
	Token  string

	// SeedProfile selects the serverHome fixture set (set by leaf Setup).
	SeedProfile string
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

func Run(t *testing.T, req *Request) (*Response, error) {
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

	moduleRoot := findModuleRoot()
	cacheDir := sessionCacheDir()
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	serverHome, err := os.MkdirTemp("", "machine-analyse-server-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(serverHome) })
	resp.ServerHome = serverHome

	if err := seedAnalyseServerHome(t, serverHome, req.SeedProfile); err != nil {
		return nil, err
	}

	agentHome, err := os.MkdirTemp("", "machine-analyse-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

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
	serverCmd.Env = append(serverCmd.Env, "HOME="+serverHome)
	serverCmd.Env = append(serverCmd.Env, "AI_CRITIC_NO_OPEN_BROWSER=1")
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
	if err := verifyServerHome(t, normalizedServer, req.Token, serverHome); err != nil {
		return nil, err
	}

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	argv := make([]string, 0, len(req.Args)+4)
	argv = append(argv, "--server", serverURL, "--token", req.Token)
	argv = append(argv, req.Args...)

	t.Logf("remote-agent argv: %v", argv)

	exitCode, stdout, stderr, runErr := runAgent(agentBin, argv, agentEnv)
	if runErr != nil {
		return nil, runErr
	}

	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	return resp, nil
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

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found")
		}
		dir = parent
	}
}

func portBaseFromTestName(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return 28000 + (hash % 1000)
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

func normalizeAbsPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return eval, nil
}

func verifyServerHome(t *testing.T, serverURL, token, wantHome string) error {
	want, err := normalizeAbsPath(wantHome)
	if err != nil {
		return fmt.Errorf("resolve harness serverHome: %w", err)
	}
	backupURL := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/api/remote-agent/machine/backup"
	body := `{"dry_run":true,"exclude":[],"include":[]}`
	req, err := http.NewRequest(http.MethodPost, backupURL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build verify-home request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify server HOME: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read verify-home response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify server HOME status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var plan struct {
		Home string `json:"home"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return fmt.Errorf("decode backup plan for HOME verify: %w", err)
	}
	got, err := normalizeAbsPath(plan.Home)
	if err != nil {
		return fmt.Errorf("resolve server-reported HOME %q: %w", plan.Home, err)
	}
	if got != want {
		return fmt.Errorf(
			"server HOME mismatch on %s: server reports %q (normalized %q) but harness serverHome is %q (normalized %q)",
			backupURL, plan.Home, got, wantHome, want,
		)
	}
	t.Logf("verified server HOME=%s", got)
	return nil
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
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
```