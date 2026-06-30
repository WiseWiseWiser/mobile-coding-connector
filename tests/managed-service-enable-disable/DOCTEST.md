# Managed Service Enable/Disable Doctests

End-to-end tests for per-service **enable/disable**: API semantics, daemon
reconcile, boot auto-start filtering, and `remote-agent service` CLI prompts.

# DSN (Domain Specific Notion)

The harness exercises managed services against a real `ai-critic-server`
subprocess with isolated `AI_CRITIC_HOME`, seeded `services.json`, and optional
`remote-agent` CLI invocation.

**Participants**

- **ai-critic-server subprocess** — loads `services.json`, runs
  `AutoStartConfiguredServices()` at boot, and `reconcileProcesses()` on a 5s
  ticker.
- **Service definitions** — persisted rows in `{AI_CRITIC_HOME}/services.json`
  with optional `enabled` (default true when absent).
- **Service processes** — long-running commands (typically `sleep`) tracked by
  PID, `desiredRunning`, and `status`.
- **HTTP client** — authenticated `POST /api/services/{disable,enable,start,stop}`
  and `GET /api/services` for server-side leaves.
- **remote-agent subprocess** — built from `./cmd/remote-agent`; prints API
  `message` to stdout for CLI leaves.
- **Test credentials** — `lib.TestPassword` token in `server-credentials`.

**Behaviors**

- **Disable** sets `enabled=false` in `services.json` without stopping the process
  or clearing `desiredRunning`.
- **Enable** sets `enabled=true`; on a stopped service sets `desired=true` so
  the daemon starts it on the next reconcile tick (~5s), without synchronous
  `start()` in the handler.
- **AutoStartConfiguredServices** starts only definitions where `enabled != false`.
- **Manual Start/Stop** work regardless of `enabled`; manual stop on an enabled
  service keeps it stopped until Start or Enable schedules it again.
- Action responses return `{ status, message, service }` with contextual prompts.

## Version

0.0.2

## Decision Tree

```
[service enable/disable — server + CLI]
 |
 +-- disable-running/                 (GROUP)  disable while process is alive
 |    +-- keeps-process/              (LEAF)   enabled=false, PID survives, prompt
 |
 +-- disable-stopped/                  (GROUP)  disable while process is stopped
 |    +-- already-stopped/            (LEAF)   enabled=false, already-stopped prompt
 |
 +-- enable-stopped/                  (GROUP)  enable while process is stopped
 |    +-- schedules-daemon/            (LEAF)   not immediate; running after ~6s
 |
 +-- enable-running/                  (GROUP)  enable while process is running
 |    +-- already-running/            (LEAF)   still running, already-running prompt
 |
 +-- autostart-skips-disabled/        (LEAF)   boot auto-start skips disabled defs
 |
 +-- cli-disable/                     (GROUP)  remote-agent service disable
 |    +-- prints-message/             (LEAF)   exit 0, stdout contains prompt
 |
 +-- cli-enable/                      (GROUP)  remote-agent service enable
      +-- prints-message/             (LEAF)   exit 0, stdout contains prompt
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `disable-running/keeps-process` | Running `sleep` service → disable keeps PID alive and returns won't-stop prompt |
| 2 | `disable-stopped/already-stopped` | Stopped service → disable persists `enabled=false` and returns already-stopped prompt |
| 3 | `enable-stopped/schedules-daemon` | Disabled stopped service → enable schedules daemon start within one reconcile window |
| 4 | `enable-running/already-running` | Disabled but manually started service → enable returns already-running prompt |
| 5 | `autostart-skips-disabled` | Mixed `services.json` at boot → only enabled services auto-start |
| 6 | `cli-disable/prints-message` | `remote-agent service disable <name>` prints contextual message, exit 0 |
| 7 | `cli-enable/prints-message` | `remote-agent service enable <name>` prints contextual message, exit 0 |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Action: disable | disable-running/*, disable-stopped/*, cli-disable/* |
| Action: enable | enable-stopped/*, enable-running/*, cli-enable/* |
| Service running at action time | disable-running, enable-running, cli-disable (running setup) |
| Service stopped at action time | disable-stopped, enable-stopped, cli-enable (stopped setup) |
| Daemon deferred start | enable-stopped/schedules-daemon |
| Boot auto-start filter | autostart-skips-disabled |
| API vs CLI invocation | disable/enable grouping leaves vs cli-* leaves |
| `enabled` default true (field absent) | autostart-skips-disabled (enabled-svc seed) |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/managed-service-enable-disable
doctest test ./tests/managed-service-enable-disable/...
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

// ServiceSeed is one row written to services.json before the server starts.
type ServiceSeed struct {
	ID      string
	Name    string
	Command string
	Enabled *bool
}

type Request struct {
	Services []ServiceSeed
	TargetID string
	Action   string
	UseCLI   bool
	CLIArgs  []string

	PreStartID string
	Token      string
	ServerPort int
	WaitAfterSecs int
}

type serviceStatus struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Command        string `json:"command"`
	Status         string `json:"status"`
	PID            int    `json:"pid"`
	DesiredRunning bool   `json:"desiredRunning"`
	Enabled        bool   `json:"enabled"`
}

type serviceActionResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Service serviceStatus `json:"service"`
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ConfigHome string
	AgentHome  string

	ActionResult           *serviceActionResponse
	ActionError            string
	ServicesAfterAction    []serviceStatus
	ServicesOnDisk         []map[string]any
	TargetPID              int
	TargetRunningImmediate bool
	TargetRunningAfterWait bool
	TargetEnabledOnDisk    *bool
}

type servicesFileRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	Enabled   *bool  `json:"enabled,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if req.TargetID == "" && len(req.Services) > 0 {
		req.TargetID = req.Services[0].ID
	}
	if req.WaitAfterSecs <= 0 && req.Action == "enable" {
		req.WaitAfterSecs = 7
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-server-enable-disable-"+safeName)
	agentBin := filepath.Join(os.TempDir(), "remote-agent-enable-disable-"+safeName)

	for _, spec := range []struct {
		out string
		pkg string
	}{
		{serverBin, "."},
		{agentBin, "./cmd/remote-agent"},
	} {
		cmd := exec.Command("go", "build", "-o", spec.out, spec.pkg)
		cmd.Dir = moduleRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
		}
		t.Cleanup(func() { os.Remove(spec.out) })
	}

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	agentHome, err := os.MkdirTemp("", "remote-agent-enable-disable-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome
	aiCriticAgent := filepath.Join(agentHome, ".ai-critic")
	if err := os.MkdirAll(aiCriticAgent, 0755); err != nil {
		return nil, err
	}

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	if len(req.Services) > 0 {
		if err := writeServicesJSON(configHome, req.Services); err != nil {
			return nil, err
		}
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort, err := pickFreePort(portBase)
	if err != nil {
		return nil, err
	}
	resp.ServerPort = serverPort

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

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	if err := waitHTTPReady(baseURL+"/ping", 30*time.Second); err != nil {
		return nil, err
	}

	if req.PreStartID != "" {
		if _, err := postServiceAction(baseURL, req.Token, "/api/services/start", req.PreStartID); err != nil {
			return nil, fmt.Errorf("pre-start %s: %w", req.PreStartID, err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	switch req.Action {
	case "", "boot-only":
		// autostart-only: server boot already ran AutoStartConfiguredServices
		if req.WaitAfterSecs > 0 {
			time.Sleep(time.Duration(req.WaitAfterSecs) * time.Second)
		}
	case "disable", "enable", "start", "stop":
		if req.UseCLI {
			serverURL := fmt.Sprintf("http://localhost:%d", serverPort)
			if err := writeRemoteAgentConfig(filepath.Join(aiCriticAgent, "remote-agent-config.json"), serverURL, req.Token); err != nil {
				return nil, err
			}
			argv := req.CLIArgs
			if len(argv) == 0 {
				argv = []string{"service", req.Action, serviceNameForTarget(req)}
			}
			fullArgv := append([]string{"--server", serverURL, "--token", req.Token}, argv...)
			agentCmd := exec.Command(agentBin, fullArgv...)
			agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
			agentEnv = append(agentEnv, "HOME="+agentHome)
			agentCmd.Env = agentEnv
			var stdout, stderr bytes.Buffer
			agentCmd.Stdout = &stdout
			agentCmd.Stderr = &stderr
			runErr := agentCmd.Run()
			if runErr != nil {
				if exitErr, ok := runErr.(*exec.ExitError); ok {
					resp.ExitCode = exitErr.ExitCode()
				} else {
					resp.ActionError = runErr.Error()
				}
			}
			resp.Stdout = stdout.String()
			resp.Stderr = stderr.String()
			resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
		} else {
			path := "/api/services/" + req.Action
			actionResp, err := postServiceAction(baseURL, req.Token, path, req.TargetID)
			if err != nil {
				resp.ActionError = err.Error()
			} else {
				resp.ActionResult = actionResp
			}
		}
	default:
		return nil, fmt.Errorf("unknown action %q", req.Action)
	}

	services, err := getServices(baseURL, req.Token)
	if err != nil {
		return nil, err
	}
	if req.TargetID != "" {
		for _, svc := range services {
			if svc.ID == req.TargetID {
				resp.TargetPID = svc.PID
				resp.TargetRunningImmediate = serviceIsRunning(svc)
				break
			}
		}
	}

	if req.WaitAfterSecs > 0 && req.Action != "" && req.Action != "boot-only" {
		time.Sleep(time.Duration(req.WaitAfterSecs) * time.Second)
		services, err = getServices(baseURL, req.Token)
		if err != nil {
			return nil, err
		}
	}

	resp.ServicesAfterAction = services

	onDisk, err := readServicesJSON(configHome)
	if err != nil {
		return nil, err
	}
	resp.ServicesOnDisk = onDisk

	if req.TargetID != "" {
		for _, svc := range services {
			if svc.ID == req.TargetID {
				resp.TargetPID = svc.PID
				resp.TargetRunningAfterWait = serviceIsRunning(svc)
				break
			}
		}
		for _, row := range onDisk {
			id, _ := row["id"].(string)
			if id == req.TargetID {
				if enabled, ok := row["enabled"].(bool); ok {
					v := enabled
					resp.TargetEnabledOnDisk = &v
				}
				break
			}
		}
	}

	return resp, nil
}
```