# Local-Agent CLI Doctests

End-to-end tests for the `local-agent` binary: server URL resolution (`--server`,
`--port`, saved config, built-in default), local reachability hints, isolated
`local-agent-config.json`, and command parity with `remote-agent`.

# DSN (Domain Specific Notion)

The harness exercises the local profile of the shared agent CLI against a real
`ai-critic-server` subprocess when needed.

**Participants**

- **local-agent subprocess** — built from `./cmd/local-agent`; parses global
  `--server`, `--port`, `--token`, then dispatches subcommands.
- **ai-critic-server subprocess** — optional; bound to an ephemeral port with
  test credentials when a leaf needs a listening API.
- **Isolated user HOME** — temp directory so `~/.ai-critic/local-agent-config.json`
  (and optional `remote-agent-config.json` sentinel) never touch the developer machine.
- **agentcli test hooks** — `AGENTCLI_TEST_DEFAULT_PORT` and
  `AGENTCLI_TEST_REACHABILITY` env vars (read by `cmd/agentcli/testhooks` at
  process start) so tests never depend on port 23712 being free and can force
  “not listening” without racey firewall tricks.

**Behaviors**

- `--server` and `--port` are mutually exclusive; both set → usage error before network.
- `--port N` resolves to `http://localhost:N` and targets that server.
- With no flags and empty config, resolution uses injected default port
  (`http://localhost:<injected>`), not a hard-coded 23712 in tests.
- Saved token in `local-agent-config.json` is used when `--server` matches a domain entry.
- When the resolved server is not listening, stderr includes an `ai-critic` start hint.
- When the server listens but auth fails, errors do not include that hint.
- `local-agent` reads/writes only `local-agent-config.json`, not `remote-agent-config.json`.

## Version

0.0.2

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
 |
 +-- config-isolation/
 |    |
 |    +-- separate-config-file/                (LEAF)  remote-agent-config.json untouched
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
| 7 | `config-isolation/separate-config-file` | `remote-agent-config.json` bytes unchanged |
| 8 | `command-parity/request-ping` | `request /ping` prints `pong` |
| 9 | `command-parity/help-branding` | Top-level help documents local branding and `--port` |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| `--server` vs `--port` conflict | port-and-server-mutually-exclusive |
| `--port` shorthand | port-shorthand |
| Built-in default (hooked) | ping-without-flags |
| Saved config token | saved-token-from-config |
| Reachability mock down | prompts-ai-critic |
| Reachability up + auth fail | auth-failure-no-hint |
| Config file boundary | separate-config-file |
| `request` subcommand | request-ping |
| Help text | help-branding |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/local-agent
doctest test ./tests/local-agent/...
```

```go
import (
	"bytes"
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

	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/script/lib"
)

// DomainEntry mirrors local-agent-config.json domain rows.
type DomainEntry struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

// LocalAgentConfigFile is the persisted JSON shape for local-agent.
type LocalAgentConfigFile struct {
	Default string        `json:"default,omitempty"`
	Domains []DomainEntry `json:"domains"`
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

	// GlobalHelp runs `local-agent -h` (top-level help, no subcommand).
	GlobalHelp bool
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
	RemoteConfigBefore []byte
	RemoteConfigAfter  []byte
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-server-local-agent-"+safeName)
	agentBin := filepath.Join(os.TempDir(), "local-agent-doctest-"+safeName)

	for _, spec := range []struct {
		out string
		pkg string
	}{
		{serverBin, "."},
		{agentBin, "./cmd/local-agent"},
	} {
		cmd := exec.Command("go", "build", "-o", spec.out, spec.pkg)
		cmd.Dir = moduleRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
		}
		t.Cleanup(func() { os.Remove(spec.out) })
	}

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

	if req.SeedRemoteConfig != nil {
		if err := os.WriteFile(resp.RemoteConfigPath, req.SeedRemoteConfig, 0600); err != nil {
			return nil, err
		}
	}
	if req.WatchRemoteConfig {
		resp.RemoteConfigBefore, _ = os.ReadFile(resp.RemoteConfigPath)
	}

	if req.SeedLocalConfig != nil {
		data, err := json.MarshalIndent(req.SeedLocalConfig, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(resp.LocalConfigPath, data, 0600); err != nil {
			return nil, err
		}
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
				return nil, err
			}
			if err := os.WriteFile(resp.LocalConfigPath, data, 0600); err != nil {
				return nil, err
			}
		}
	}

	argv := buildAgentArgv(req)
	t.Logf("local-agent argv: %v", argv)

	agentCmd := exec.Command(agentBin, argv...)
	agentEnv := append([]string{}, os.Environ()...)
	agentEnv = append(agentEnv, "HOME="+agentHome)
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

	return resp, nil
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

func findModuleRoot() (string, error) {
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

func pickFreePort(base int) (int, error) {
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