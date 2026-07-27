# Managed Service Enable/Disable Doctests

End-to-end tests for per-service **enable/disable**: API semantics, daemon
reconcile, boot auto-start filtering, and `remote-agent service` CLI prompts.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `services.NewManagerAt` + Manager methods
(or `agentcli.Run` against `RegisterAPIWithManager`). One sparse **L3 e2e** smoke
keeps the product binary path (`UseBinary` + `label: heavy, e2e`):
`disable-running/keeps-process`.

**Participants**

- **L2: services.Manager** — isolated config dir; Enable/Disable/Start/AutoStart.
- **L2: agentcli.Run** — in-process `service enable|disable` against local mux.
- **L3: ai-critic-server + remote-agent subprocesses** — sparse binary smoke.
- **Service definitions** — `services.json` with optional `enabled`.
- **Service processes** — long-running `sleep` tracked by PID / desiredRunning.

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

0.0.3

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
	"net/http/httptest"
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
	"github.com/xhd2015/ai-critic/server/services"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run (package-level state).
var agentcliInProcessMu sync.Mutex


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
	UseCLI   bool // invoke via agentcli/CLI path (L2 or L3)
	CLIArgs  []string

	// UseBinary / E2E force L3 product binaries. Default false → L2 Manager.
	UseBinary bool
	E2E       bool

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

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseBinary || req.E2E)
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if req.TargetID == "" && len(req.Services) > 0 {
		req.TargetID = req.Services[0].ID
	}
	if req.WaitAfterSecs <= 0 && req.Action == "enable" && !req.UseCLI {
		req.WaitAfterSecs = 7
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

	if len(req.Services) > 0 {
		if err := writeServicesJSON(configHome, req.Services); err != nil {
			return nil, err
		}
	}

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, configHome, agentHome, aiCriticAgent)
	}
	return runInProcessL2(t, req, resp, configHome, agentHome, aiCriticAgent)
}

func runInProcessL2(t *testing.T, req *Request, resp *Response, configHome, agentHome, aiCriticAgent string) (*Response, error) {
	m := services.NewManagerAt(configHome)
	t.Cleanup(func() { m.Shutdown() })

	// Boot auto-start for boot-only leaves.
	if req.Action == "" || req.Action == "boot-only" {
		go m.AutoStartConfiguredServices()
		if req.WaitAfterSecs > 0 {
			time.Sleep(time.Duration(req.WaitAfterSecs) * time.Second)
		} else {
			time.Sleep(3 * time.Second)
		}
		if err := fillServiceSnapshot(req, resp, m, configHome); err != nil {
			return nil, err
		}
		return resp, nil
	}

	if req.PreStartID != "" {
		if _, err := m.Start(req.PreStartID); err != nil {
			return nil, fmt.Errorf("pre-start %s: %w", req.PreStartID, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Enable deferred start needs health reconcile.
	if req.Action == "enable" {
		m.StartHealthCheck()
	}

	if req.UseCLI {
		mux := http.NewServeMux()
		services.RegisterAPIWithManager(mux, m)
		mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/ping" {
				mux.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+req.Token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			mux.ServeHTTP(w, r)
		})
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)
		serverURL := srv.URL
		if err := writeRemoteAgentConfig(filepath.Join(aiCriticAgent, "remote-agent-config.json"), serverURL, req.Token); err != nil {
			return nil, err
		}
		argv := req.CLIArgs
		if len(argv) == 0 {
			argv = []string{"service", req.Action, serviceNameForTarget(req)}
		}
		fullArgv := append([]string{"--server", serverURL, "--token", req.Token}, argv...)
		exitCode, stdout, stderr, runErr := runAgentcliCaptured(fullArgv)
		resp.ExitCode = exitCode
		resp.Stdout = stdout
		resp.Stderr = stderr
		resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)
		if runErr != nil && resp.ActionError == "" {
			resp.ActionError = runErr.Error()
		}
	} else {
		switch req.Action {
		case "disable":
			ar, err := m.Disable(req.TargetID)
			if err != nil {
				resp.ActionError = err.Error()
			} else {
				resp.ActionResult = toActionResponse(ar)
			}
		case "enable":
			ar, err := m.Enable(req.TargetID)
			if err != nil {
				resp.ActionError = err.Error()
			} else {
				resp.ActionResult = toActionResponse(ar)
			}
		case "start":
			st, err := m.Start(req.TargetID)
			if err != nil {
				resp.ActionError = err.Error()
			} else if st != nil {
				resp.ActionResult = &serviceActionResponse{Status: "ok", Service: toServiceStatus(st)}
			}
		case "stop":
			// Manager has no public Stop; use Restart reverse via API path not available.
			// stop via package: list then disable doesn't stop. Use binary for stop only if needed.
			// For enable/disable trees we only need start as PreStart.
			resp.ActionError = "stop not used in L2 path"
		default:
			return nil, fmt.Errorf("unknown action %q", req.Action)
		}
	}

	// Immediate snapshot
	if err := fillServiceSnapshot(req, resp, m, configHome); err != nil {
		return nil, err
	}
	resp.TargetRunningImmediate = resp.TargetRunningAfterWait

	if req.WaitAfterSecs > 0 && req.Action != "" && req.Action != "boot-only" {
		time.Sleep(time.Duration(req.WaitAfterSecs) * time.Second)
		if err := fillServiceSnapshot(req, resp, m, configHome); err != nil {
			return nil, err
		}
	}
	t.Logf("L2 Manager action=%s target=%s", req.Action, req.TargetID)
	return resp, nil
}

func findListed(m *services.Manager, id string) (services.ServiceStatus, bool) {
	for _, s := range m.ListAll() {
		if s.ID == id {
			return s, true
		}
	}
	return services.ServiceStatus{}, false
}

func toServiceStatus(st *services.ServiceStatus) serviceStatus {
	if st == nil {
		return serviceStatus{}
	}
	return serviceStatus{
		ID: st.ID, Name: st.Name, Command: st.Command, Status: st.Status,
		PID: st.PID, DesiredRunning: st.DesiredRunning, Enabled: st.Enabled,
	}
}

func toActionResponse(ar *services.ServiceActionResponse) *serviceActionResponse {
	if ar == nil {
		return nil
	}
	out := &serviceActionResponse{Status: ar.Status, Message: ar.Message}
	if ar.Service != nil {
		out.Service = toServiceStatus(ar.Service)
	}
	return out
}

func fillServiceSnapshot(req *Request, resp *Response, m *services.Manager, configHome string) error {
	listed := m.ListAll()
	svcs := make([]serviceStatus, 0, len(listed))
	for _, s := range listed {
		svcs = append(svcs, toServiceStatus(&s))
	}
	resp.ServicesAfterAction = svcs
	if req.TargetID != "" {
		for _, svc := range svcs {
			if svc.ID == req.TargetID {
				resp.TargetPID = svc.PID
				resp.TargetRunningAfterWait = serviceIsRunning(svc)
				break
			}
		}
	}
	onDisk, err := readServicesJSON(configHome)
	if err != nil {
		return err
	}
	resp.ServicesOnDisk = onDisk
	if req.TargetID != "" {
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
	return nil
}


func runAgentcliCaptured(argv []string) (exitCode int, stdout, stderr string, err error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return 1, "", "", err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return 1, "", "", err
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

	exitCode = 0
	if runErr != nil {
		exitCode = 1
	}
	return exitCode, string(outBytes), string(errBytes), runErr
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, configHome, agentHome, aiCriticAgent string) (*Response, error) {
	moduleRoot, err := findModuleRoot(d)
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

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort, err := pickFreePort(portBase)
	if err != nil {
		return nil, err
	}
	resp.ServerPort = serverPort

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = configHome
	env := lib.AppendTestServerEnv(os.Environ(), configHome)
	env = stripEnvPrefix(env, "AI_CRITIC_TEST_SKIP_EXTENSION=")
	env = append(env, "AI_CRITIC_TEST_SKIP_EXTENSION=0")
	serverCmd.Env = env
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

	servicesList, err := getServices(baseURL, req.Token)
	if err != nil {
		return nil, err
	}
	if req.TargetID != "" {
		for _, svc := range servicesList {
			if svc.ID == req.TargetID {
				resp.TargetPID = svc.PID
				resp.TargetRunningImmediate = serviceIsRunning(svc)
				break
			}
		}
	}

	if req.WaitAfterSecs > 0 && req.Action != "" && req.Action != "boot-only" {
		time.Sleep(time.Duration(req.WaitAfterSecs) * time.Second)
		servicesList, err = getServices(baseURL, req.Token)
		if err != nil {
			return nil, err
		}
	}

	resp.ServicesAfterAction = servicesList

	onDisk, err := readServicesJSON(configHome)
	if err != nil {
		return nil, err
	}
	resp.ServicesOnDisk = onDisk

	if req.TargetID != "" {
		for _, svc := range servicesList {
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