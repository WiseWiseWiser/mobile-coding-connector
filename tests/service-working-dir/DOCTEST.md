# Service Working Directory Doctests

End-to-end tests for managed service `workingDir` auto-creation: when the
configured directory is missing on disk, `POST /api/services/start` must create
it (including parents) before launching the shell command.

# DSN (Domain Specific Notion)

The harness exercises managed services against a real `ai-critic-server`
subprocess with isolated `AI_CRITIC_HOME`, seeded `services.json` rows that
include `workingDir`, and HTTP start/status checks.

**Participants**

- **ai-critic-server subprocess** — loads `services.json`, runs
  `ensureServiceWorkingDir` before `exec.Command`, and tracks service PIDs.
- **Service definitions** — persisted rows in `{AI_CRITIC_HOME}/services.json`
  with `id`, `name`, `command`, and optional `workingDir`.
- **Service processes** — long-running `sleep 300` commands launched with
  `cmd.Dir` set to `workingDir`.
- **Service log** — append-only file at `{AI_CRITIC_HOME}/services/{id}.log`
  with start markers and failure lines.
- **HTTP client** — authenticated `POST /api/services/start` and
  `GET /api/services` for status and PID.
- **Test credentials** — `lib.TestPassword` token in `server-credentials`.

**Behaviors**

- **Missing workingDir** — `os.MkdirAll` creates the path (including nested
  parents) before `bash -lc` runs; start succeeds with `status=running` and
  `pid>0`.
- **Missing workingDir without fix** — `cmd.Start()` fails with misleading
  `fork/exec /bin/bash: no such file or directory` when `cmd.Dir` is absent.
- **Existing workingDir** — no error; service starts normally with the same
  running checks.
- **Start API** — returns `{ status, message, service }` on success.

## Version

0.0.2

## Decision Tree

```
[service working dir on start]
 |
 +-- missing-dir/                    (GROUP)  workingDir absent on disk
 |    +-- creates-and-runs/          (LEAF)   mkdir → start succeeds, pid > 0
 |    +-- no-bash-fork-error/        (LEAF)   log lacks fork/exec /bin/bash
 |    +-- nested-path/               (LEAF)   deep nested dir created
 |
 +-- existing-dir/                   (GROUP)  workingDir already exists
      +-- start-unchanged/           (LEAF)   service still starts normally
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `missing-dir/creates-and-runs` | Missing flat `workingDir` → created on disk, service running with `pid>0` |
| 2 | `missing-dir/no-bash-fork-error` | Service log contains `starting service`, not `fork/exec /bin/bash` |
| 3 | `missing-dir/nested-path` | Missing nested `a/b/c` path → created, service running |
| 4 | `existing-dir/start-unchanged` | Pre-created `workingDir` → start succeeds unchanged |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| `workingDir` missing on disk | missing-dir/* |
| `workingDir` pre-existing | existing-dir/start-unchanged |
| Flat vs nested path depth | creates-and-runs vs nested-path |
| Log error surface | no-bash-fork-error |
| Disk creation verified | creates-and-runs, nested-path |
| PID / status running | all leaves |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/service-working-dir
doctest test ./tests/service-working-dir/...
```

```go
import (
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

func Run(t *testing.T, req *Request) (*Response, error) {
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

	moduleRoot, err := findModuleRoot()
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

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

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

	startResult, err := postServiceStart(baseURL, req.Token, req.TargetID)
	if err != nil {
		resp.StartError = err.Error()
	} else {
		resp.StartResult = startResult
	}

	time.Sleep(500 * time.Millisecond)

	services, err := getServices(baseURL, req.Token)
	if err != nil {
		return nil, err
	}
	resp.ServicesAfterStart = services

	for _, svc := range services {
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