# Service Working Directory Doctests

Managed service `workingDir` auto-creation: when the configured directory is
missing on disk, start must create it (including parents) before launching the
shell command.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `services.NewManagerAt` + `Manager.Start`
(isolated config dir; no product binary). One sparse **L3 e2e** smoke keeps the
product binary path (`UseCLI` + `label: heavy, e2e`): `missing-dir/creates-and-runs`.

**Participants**

- **L2: services.Manager** — `NewManagerAt(configHome)` + `Start` after seeded JSON.
- **L3: ai-critic-server subprocess** — sparse `UseCLI` smoke with HTTP start.
- **Service definitions** — `workingDir` rows; long-running `sleep 300` for PID checks.
- **Service log** — `{configHome}/services/{id}.log` with start markers.

**Behaviors**

- **Missing workingDir** — `os.MkdirAll` creates the path before `bash -lc` runs.
- **Existing workingDir** — start succeeds unchanged.
- Log contains `starting service`, not `fork/exec /bin/bash`.

## Version

0.0.3

## Decision Tree

```
[service working dir on start]
 |
 +-- missing-dir/                    (GROUP)  workingDir absent on disk
 |    +-- creates-and-runs/          (LEAF)   mkdir → start succeeds, pid > 0  [L3 smoke]
 |    +-- no-bash-fork-error/        (LEAF)   log lacks fork/exec /bin/bash
 |    +-- nested-path/               (LEAF)   deep nested dir created
 |
 +-- existing-dir/                   (GROUP)  workingDir already exists
      +-- start-unchanged/           (LEAF)   service still starts normally
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `missing-dir/creates-and-runs` | L3 smoke: missing flat workingDir → created, pid>0 |
| 2 | `missing-dir/no-bash-fork-error` | L2: log lacks fork/exec /bin/bash |
| 3 | `missing-dir/nested-path` | L2: nested path created |
| 4 | `existing-dir/start-unchanged` | L2: pre-created workingDir still starts |

## How to Run

```sh
doctest vet ./tests/service-working-dir
doctest test ./tests/service-working-dir/...
doctest test --label e2e ./tests/service-working-dir/...  # 1 L3 smoke
```

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/services"
	"github.com/xhd2015/doctest/session"
)

// ServiceSeed is one row written to services.json before the server starts.
type ServiceSeed struct {
	ID         string
	Name       string
	Command    string
	WorkingDir string
}

type Request struct {
	Services   []ServiceSeed
	TargetID   string
	WorkingDir string
	TempBase   string
	Token      string
	ServerPort int

	// UseCLI forces L3 product-binary path. Default false → L2 Manager.
	UseCLI bool
	E2E    bool
}

type serviceStatus struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Command        string `json:"command"`
	WorkingDir     string `json:"workingDir,omitempty"`
	Status         string `json:"status"`
	PID            int    `json:"pid"`
	DesiredRunning bool   `json:"desiredRunning"`
}

type Response struct {
	ServerPort         int
	ConfigHome         string
	WorkingDir         string
	StartResult        *serviceStatus
	StartError         string
	ServicesAfterStart []serviceStatus
	TargetPID          int
	TargetRunning      bool
	WorkingDirExists   bool
	WorkingDirIsDir    bool
	ServiceLog         string
}

type servicesFileRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	WorkingDir string `json:"workingDir,omitempty"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseCLI || req.E2E)
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if req.TargetID == "" && len(req.Services) > 0 {
		req.TargetID = req.Services[0].ID
	}
	if req.WorkingDir == "" && len(req.Services) > 0 {
		req.WorkingDir = req.Services[0].WorkingDir
	}
	resp.WorkingDir = req.WorkingDir

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	if len(req.Services) > 0 {
		if err := writeServicesJSON(configHome, req.Services); err != nil {
			return nil, err
		}
	}

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, configHome)
	}
	return runInProcessL2(t, req, resp, configHome)
}

func runInProcessL2(t *testing.T, req *Request, resp *Response, configHome string) (*Response, error) {
	m := services.NewManagerAt(configHome)
	t.Cleanup(func() { m.Shutdown() })

	st, err := m.Start(req.TargetID)
	if err != nil {
		resp.StartError = err.Error()
	} else if st != nil {
		resp.StartResult = &serviceStatus{
			ID:             st.ID,
			Name:           st.Name,
			Command:        st.Command,
			WorkingDir:     st.WorkingDir,
			Status:         st.Status,
			PID:            st.PID,
			DesiredRunning: st.DesiredRunning,
		}
	}

	time.Sleep(300 * time.Millisecond)

	listed := m.ListAll()
	after := make([]serviceStatus, 0, len(listed))
	for _, svc := range listed {
		after = append(after, serviceStatus{
			ID:             svc.ID,
			Name:           svc.Name,
			Command:        svc.Command,
			WorkingDir:     svc.WorkingDir,
			Status:         svc.Status,
			PID:            svc.PID,
			DesiredRunning: svc.DesiredRunning,
		})
		if svc.ID == req.TargetID {
			resp.TargetPID = svc.PID
			resp.TargetRunning = svc.Status == "running" || svc.Status == "starting"
		}
	}
	resp.ServicesAfterStart = after

	if req.WorkingDir != "" {
		info, statErr := os.Stat(req.WorkingDir)
		if statErr == nil {
			resp.WorkingDirExists = true
			resp.WorkingDirIsDir = info.IsDir()
		}
	}

	logPath := filepath.Join(configHome, "services", req.TargetID+".log")
	if data, readErr := os.ReadFile(logPath); readErr == nil {
		resp.ServiceLog = string(data)
	}
	t.Logf("L2 Manager.Start workingDir=%s pid=%d", req.WorkingDir, resp.TargetPID)
	return resp, nil
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, configHome string) (*Response, error) {
	moduleRoot, err := findModuleRoot(d)
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-server-working-dir-"+safeName)
	build := exec.Command("go", "build", "-o", serverBin, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build server: %w\n%s", err, string(out))
	}
	t.Cleanup(func() { os.Remove(serverBin) })

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

	startResult, err := postServiceStart(baseURL, req.Token, req.TargetID)
	if err != nil {
		resp.StartError = err.Error()
	} else {
		// postServiceStart in SETUP may decode top-level or nested service fields.
		resp.StartResult = startResult
	}

	time.Sleep(500 * time.Millisecond)

	svcs, err := getServices(baseURL, req.Token)
	if err != nil {
		return nil, err
	}
	resp.ServicesAfterStart = svcs

	for _, svc := range svcs {
		if svc.ID == req.TargetID {
			resp.TargetPID = svc.PID
			resp.TargetRunning = serviceIsRunning(svc)
			break
		}
	}

	if req.WorkingDir != "" {
		info, statErr := os.Stat(req.WorkingDir)
		if statErr == nil {
			resp.WorkingDirExists = true
			resp.WorkingDirIsDir = info.IsDir()
		}
	}

	logPath := filepath.Join(configHome, "services", req.TargetID+".log")
	if data, readErr := os.ReadFile(logPath); readErr == nil {
		resp.ServiceLog = string(data)
	}
	return resp, nil
}
```
