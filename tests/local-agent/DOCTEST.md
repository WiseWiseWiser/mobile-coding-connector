# Local-Agent CLI Doctests

Scenario tests for the `local-agent` CLI: server URL resolution (`--server`,
`--port`, saved config, built-in default), local reachability hints, isolated
`local-agent-config.json`, and command parity with `remote-agent`.

Most leaves are **L2 in-process** (`agentcli.Run(LocalProfile())` with testhooks
home/port/reachability overrides and optional in-process `/ping` + auth stubs).
Two sparse **L3 e2e** smokes keep the product binary path: `not-running/prompts-ai-critic`
and `command-parity/help-branding` (less-gen `os.Exit` on help).

# DSN (Domain Specific Notion)

**Participants**

- **L2: agentcli.Run** — `LocalProfile()` under a suite mutex; home/port/reachability
  via `testhooks` package vars (no process Setenv).
- **L2: optional httptest mux** — `/ping` + `/api/auth/status` + `/api/projects` stubs
  when `StartServer` is set (no product server binary).
- **L3: local-agent subprocess** — `UseCLI` smokes only (`./cmd/local-agent`).
- **L3: ai-critic-server subprocess** — only when a UseCLI leaf also needs a real server.
- **Isolated user HOME** — temp dir via testhooks override (L2) or child `HOME=` (L3).
- **agentcli test hooks** — in-process setters or child env for default port / reachability.

**Behaviors**

- `--server` and `--port` are mutually exclusive; both set → usage error before network.
- `--port N` resolves to `http://localhost:N` and targets that server.
- With no flags and empty config, resolution uses injected default port
  (`http://localhost:<injected>`), not a hard-coded 23712 in tests.
- Saved token in `local-agent-config.json` is used when `--server` matches a domain entry.
- When the resolved server is not listening, stderr includes an `ai-critic` start hint.
- When the server listens but auth fails, errors do not include that hint.
- `local-agent` reads/writes only `local-agent-config.json`, not `remote-agent-config.json`.
- Auth failures include profile-specific CLI authorization guidance.
- `local-agent auth import-local` reads the first non-empty local server credential
  from `~/.ai-critic/server-credentials`, stores it for the resolved local server,
  preserves unrelated config fields, and never prints the raw token.

## Version

0.0.3

## Decision Tree

```
[local-agent CLI]
 |
 +-- flags/
 |    |
 |    +-- port-and-server-mutually-exclusive/   (LEAF)  --port + --server → usage error
 |    +-- port-shorthand/                       (LEAF)  --port N → http://localhost:N
 |
 +-- default-resolution/
 |    |
 |    +-- ping-without-flags/                  (LEAF)  default URL → injected port, pong
 |    +-- saved-token-from-config/             (LEAF)  token from local-agent-config.json
 |
 +-- not-running/
 |    |
 |    +-- prompts-ai-critic/                   (LEAF)  not listening → stderr mentions ai-critic
 |    +-- auth-failure-no-hint/                (LEAF)  listening, bad token → no ai-critic hint
 |    +-- project-list-auth-guidance/          (LEAF)  project list bad token → local config + credential hint
 |
 +-- config-isolation/
 |    |
 |    +-- separate-config-file/                (LEAF)  remote-agent-config.json untouched
 |
 +-- auth-import-local/
 |    |
 |    +-- first-non-empty-token/               (LEAF)  local credential imported without printing token
 |
 +-- command-parity/
      |
      +-- request-ping/                        (LEAF)  request /ping → pong
      +-- help-branding/                       (LEAF)  help names local-agent, --port, 23712
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `flags/port-and-server-mutually-exclusive` | Usage error when `--port` and `--server` are both set |
| 2 | `flags/port-shorthand` | `--port` targets `http://localhost:N` with a live server |
| 3 | `default-resolution/ping-without-flags` | No flags/config uses built-in default (injected port) |
| 4 | `default-resolution/saved-token-from-config` | `auth status` uses saved token without `--token` |
| 5 | `not-running/prompts-ai-critic` | Unreachable server stderr includes start hint |
| 6 | `not-running/auth-failure-no-hint` | Bad token after reachability passes; no start hint |
| 7 | `not-running/project-list-auth-guidance` | `project list` bad token prints local auth guidance |
| 8 | `config-isolation/separate-config-file` | `remote-agent-config.json` bytes unchanged |
| 9 | `auth-import-local/first-non-empty-token` | `auth import-local` imports first non-empty local server credential |
| 10 | `command-parity/request-ping` | `request /ping` prints `pong` |
| 11 | `command-parity/help-branding` | Top-level help documents local branding and `--port` |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| `--server` vs `--port` conflict | port-and-server-mutually-exclusive |
| `--port` shorthand | port-shorthand |
| Built-in default (hooked) | ping-without-flags |
| Saved config token | saved-token-from-config |
| Reachability mock down | prompts-ai-critic |
| Reachability up + auth fail | auth-failure-no-hint, project-list-auth-guidance |
| Config file boundary | separate-config-file |
| Local credential import | auth-import-local/first-non-empty-token |
| `request` subcommand | request-ping |
| Help text | help-branding |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/local-agent
doctest test ./tests/local-agent/...          # unlabeled L2 mass
doctest test --label e2e ./tests/local-agent/...  # ~2 L3 smokes
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
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/doctest/session"
)

// localAgentInProcessMu serializes in-process agentcli.Run (active profile +
// stdout/stderr capture + testhooks overrides are process-global).
var localAgentInProcessMu sync.Mutex

// DomainEntry mirrors local-agent-config.json domain rows.
type DomainEntry struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

// ProjectBinding mirrors unrelated project_bindings rows preserved by config writes.
type ProjectBinding struct {
	Server    string `json:"server"`
	RemoteDir string `json:"remote_dir"`
	LocalPath string `json:"local_path"`
}

// LocalAgentConfigFile is the persisted JSON shape for local-agent.
type LocalAgentConfigFile struct {
	Default         string           `json:"default,omitempty"`
	Domains         []DomainEntry    `json:"domains"`
	ProjectBindings []ProjectBinding `json:"project_bindings,omitempty"`
}

type Request struct {
	// Subcommand argv after global flags (e.g. []string{"ping"}, []string{"request", "/ping"}).
	Args []string

	Server string
	Port   int
	Token  string
	// TokenSpecified mirrors CLI --token presence for resolveClient semantics.
	TokenSpecified bool

	// InjectedDefaultPort overrides built-in 23712 in the child via testhooks (0 = hook default 23712).
	InjectedDefaultPort int
	// MockReachability: nil = real TCP/ping check; true = always listening; false = always down.
	MockReachability *bool

	StartServer      bool
	ServerListenPort int
	// SyncPortFlagFromServer sets --port to the bound server port when StartServer and Port==0.
	SyncPortFlagFromServer bool
	// SyncDefaultPortFromServer sets InjectedDefaultPort to the bound server port when StartServer.
	SyncDefaultPortFromServer bool
	// SyncServerFromBoundPort sets --server to http://localhost:<bound> after server starts.
	SyncServerFromBoundPort bool
	// SeedLocalConfigAfterServer writes local-agent-config.json once server URL is known.
	SeedLocalConfigAfterServer bool
	LocalConfigToken           string

	SeedLocalConfig   *LocalAgentConfigFile
	SeedRemoteConfig  []byte
	WatchRemoteConfig bool
	WatchLocalConfig  bool
	ServerCredentialContent string

	// GlobalHelp runs `local-agent -h` (top-level help, no subcommand).
	GlobalHelp bool

	// UseCLI forces the L3 product-binary path. Default false → L2 in-process.
	UseCLI bool
	E2E    bool
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
	AgentHome  string

	LocalConfigPath    string
	RemoteConfigPath   string
	LocalConfigAfter   []byte
	RemoteConfigBefore []byte
	RemoteConfigAfter  []byte
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if useBinaryPath(req) {
		return runLocalAgentCLI(t, d, req, resp)
	}
	return runLocalAgentInProcess(t, d, req, resp)
}

func runLocalAgentInProcess(t *testing.T, d *session.Doctest, req *Request, resp *Response) (*Response, error) {
	t.Helper()

	agentHome, err := os.MkdirTemp("", "local-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	aiCriticDir := filepath.Join(agentHome, ".ai-critic")
	if err := os.MkdirAll(aiCriticDir, 0755); err != nil {
		return nil, err
	}
	resp.LocalConfigPath = filepath.Join(aiCriticDir, "local-agent-config.json")
	resp.RemoteConfigPath = filepath.Join(aiCriticDir, "remote-agent-config.json")

	if err := seedAgentFiles(req, resp); err != nil {
		return nil, err
	}

	serverPort := req.ServerListenPort
	if req.StartServer {
		if serverPort <= 0 {
			serverPort, err = pickFreePort(24700)
			if err != nil {
				return nil, err
			}
		}
		resp.ServerPort = serverPort
		if err := startInProcessLocalServer(t, req, resp, serverPort); err != nil {
			return nil, err
		}
		if err := applyServerSyncFlags(req, serverPort, resp); err != nil {
			return nil, err
		}
	}

	argv := buildAgentArgv(req)
	t.Logf("L2-inprocess local-agent argv: %v", argv)

	localAgentInProcessMu.Lock()
	defer localAgentInProcessMu.Unlock()

	testhooks.SetHomeOverride(agentHome)
	if req.InjectedDefaultPort > 0 {
		testhooks.SetDefaultPortForTest(req.InjectedDefaultPort)
	}
	if req.MockReachability != nil {
		if *req.MockReachability {
			testhooks.SetReachabilityForTest("up")
		} else {
			testhooks.SetReachabilityForTest("down")
		}
	}
	defer testhooks.ResetInProcessOverrides()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutW, stderrW

	runErr := agentcli.Run(agentcli.LocalProfile(), argv)

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr

	outBytes, _ := io.ReadAll(stdoutR)
	errBytes, _ := io.ReadAll(stderrR)
	_ = stdoutR.Close()
	_ = stderrR.Close()

	resp.Stdout = string(outBytes)
	resp.Stderr = string(errBytes)
	if runErr != nil {
		resp.Stderr += fmt.Sprintf("Error: %v\n", runErr)
		resp.ExitCode = 1
	}
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	if req.WatchRemoteConfig {
		resp.RemoteConfigAfter, _ = os.ReadFile(resp.RemoteConfigPath)
	}
	if req.WatchLocalConfig {
		resp.LocalConfigAfter, _ = os.ReadFile(resp.LocalConfigPath)
	}
	return resp, nil
}

func startInProcessLocalServer(t *testing.T, req *Request, resp *Response, serverPort int) error {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == lib.TestPassword {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"initialized": true,
				"status":      "ok",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"initialized": true,
			"status":      "unauthorized",
		})
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != lib.TestPassword {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		_ = json.NewEncoder(w).Encode([]interface{}{})
	})
	// Catch-all API auth for request subcommand variants.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != lib.TestPassword {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	if err != nil {
		return fmt.Errorf("listen in-process local-agent server: %w", err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 10*time.Second); err != nil {
		return err
	}
	t.Logf("L2 in-process local-agent server on port %d", serverPort)
	return nil
}

func seedAgentFiles(req *Request, resp *Response) error {
	if req.SeedRemoteConfig != nil {
		if err := os.WriteFile(resp.RemoteConfigPath, req.SeedRemoteConfig, 0600); err != nil {
			return err
		}
	}
	if req.ServerCredentialContent != "" {
		credPath := filepath.Join(filepath.Dir(resp.LocalConfigPath), "server-credentials")
		if err := os.WriteFile(credPath, []byte(req.ServerCredentialContent), 0600); err != nil {
			return err
		}
	}
	if req.WatchRemoteConfig {
		resp.RemoteConfigBefore, _ = os.ReadFile(resp.RemoteConfigPath)
	}
	if req.SeedLocalConfig != nil {
		data, err := json.MarshalIndent(req.SeedLocalConfig, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(resp.LocalConfigPath, data, 0600); err != nil {
			return err
		}
	}
	return nil
}

func applyServerSyncFlags(req *Request, serverPort int, resp *Response) error {
	if req.SyncPortFlagFromServer && req.Port <= 0 {
		req.Port = serverPort
	}
	if req.SyncDefaultPortFromServer {
		req.InjectedDefaultPort = serverPort
	}
	if req.SyncServerFromBoundPort {
		req.Server = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	if req.SeedLocalConfigAfterServer {
		token := req.LocalConfigToken
		if token == "" {
			token = lib.TestPassword
		}
		serverURL := fmt.Sprintf("http://localhost:%d", serverPort)
		cfg := &LocalAgentConfigFile{
			Default: serverURL,
			Domains: []DomainEntry{{Server: serverURL, Token: token}},
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(resp.LocalConfigPath, data, 0600); err != nil {
			return err
		}
	}
	return nil
}

func runLocalAgentCLI(t *testing.T, d *session.Doctest, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	agentHome, err := os.MkdirTemp("", "local-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	aiCriticDir := filepath.Join(agentHome, ".ai-critic")
	if err := os.MkdirAll(aiCriticDir, 0755); err != nil {
		return nil, err
	}
	resp.LocalConfigPath = filepath.Join(aiCriticDir, "local-agent-config.json")
	resp.RemoteConfigPath = filepath.Join(aiCriticDir, "remote-agent-config.json")

	if err := seedAgentFiles(req, resp); err != nil {
		return nil, err
	}

	serverPort := req.ServerListenPort
	if req.StartServer {
		if serverPort <= 0 {
			serverPort, err = pickFreePort(24700)
			if err != nil {
				return nil, err
			}
		}
		resp.ServerPort = serverPort

		configHome, err := lib.CreateTestConfigHome()
		if err != nil {
			return nil, err
		}
		t.Cleanup(func() { os.RemoveAll(configHome) })

		credFile, err := lib.WriteTestCredentials(configHome)
		if err != nil {
			return nil, err
		}

		serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
		serverCmd.Dir = configHome
		serverCmd.Env = lib.AppendTestServerEnv(os.Environ(), configHome)
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
		if err := applyServerSyncFlags(req, serverPort, resp); err != nil {
			return nil, err
		}
	}

	argv := buildAgentArgv(req)
	t.Logf("L3-binary local-agent argv: %v", argv)

	agentCmd := exec.Command(agentBin, argv...)
	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = stripEnvPrefix(agentEnv, "PATH=")
	agentEnv = append(agentEnv, "HOME="+agentHome)
	agentEnv = append(agentEnv, "PATH=/usr/bin:/bin:/usr/sbin:/sbin")
	if req.InjectedDefaultPort > 0 {
		agentEnv = testhooks.AppendDefaultPortEnv(agentEnv, req.InjectedDefaultPort)
	}
	if req.MockReachability != nil {
		agentEnv = testhooks.AppendReachabilityEnv(agentEnv, *req.MockReachability)
	}
	agentCmd.Env = agentEnv

	var stdout, stderr bytes.Buffer
	agentCmd.Stdout = &stdout
	agentCmd.Stderr = &stderr

	runErr := agentCmd.Run()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			return nil, runErr
		}
	}
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	if req.WatchRemoteConfig {
		resp.RemoteConfigAfter, _ = os.ReadFile(resp.RemoteConfigPath)
	}
	if req.WatchLocalConfig {
		resp.LocalConfigAfter, _ = os.ReadFile(resp.LocalConfigPath)
	}

	return resp, nil
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

func buildAgentArgv(req *Request) []string {
	if req.GlobalHelp {
		return []string{"-h"}
	}
	var argv []string
	if req.Server != "" {
		argv = append(argv, "--server", req.Server)
	}
	if req.Port > 0 {
		argv = append(argv, "--port", strconv.Itoa(req.Port))
	}
	if req.TokenSpecified {
		argv = append(argv, "--token", req.Token)
	}
	argv = append(argv, req.Args...)
	return argv
}

func pickFreePort(base int) (int, error) {
	// Prefer kernel-assigned free port to avoid parallel leaf races (TOCTOU).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		port := ln.Addr().(*net.TCPAddr).Port
		ln.Close()
		return port, nil
	}
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
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


```
