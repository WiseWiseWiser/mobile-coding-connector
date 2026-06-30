# Frontend Service Enable/Disable Doctests

Playwright UI tests for Disable/Enable controls on `/home/service` (`ServicesSection`).

# DSN (Domain Specific Notion)

The harness extends the frontend quick-test + Playwright pattern with API-driven
service seeding before each browser script runs.

**Participants**

- **Quick-test server** — isolated `AI_CRITIC_HOME`, Vite proxy, test credentials.
- **Service API** — `POST /api/services`, `start`, `stop`, `disable`, `enable`.
- **ServicesSection** — service cards on `/home/service` with Disable/Enable buttons
  and `ConfirmModal` prompts.
- **Playwright** — headless Chromium runs leaf `script.js`, emitting JSON for `Assert`.

**Behaviors**

- Root `Run` starts quick-test, seeds one service via API per `Request.ServiceSeed`,
  runs Playwright against `/home/service`.
- Disable on a running service shows the won't-stop-immediately modal; process stays up.
- Enable on a disabled stopped service shows the daemon-check modal; UI reflects
  `enabled` state after confirm.

## Version

0.0.2

## Decision Tree

```
[frontend service enable/disable]
 |
 +-- disable-running/                 (GROUP)  Disable while service is running
 |    +-- shows-prompt/               (LEAF)   modal prompt; service stays running
 |
 +-- enable-stopped/                  (GROUP)  Enable while disabled + stopped
      +-- shows-prompt/               (LEAF)   modal prompt; enabled badge updates
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `disable-running/shows-prompt` | Running service → Disable → modal won't-stop message; still running |
| 2 | `enable-stopped/shows-prompt` | Disabled stopped service → Enable → daemon modal; enabled UI updates |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| UI action: Disable | disable-running/shows-prompt |
| UI action: Enable | enable-stopped/shows-prompt |
| Service running | disable-running/* |
| Service stopped + disabled | enable-stopped/* |
| Route `/home/service` | both leaves |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/frontend/service-enable-disable
doctest test --label ui-automation ./tests/frontend/service-enable-disable/...
```

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	envpkg "github.com/xhd2015/ai-critic/server/env"
)

type ServiceSeed struct {
	ID               string
	Name             string
	Command          string
	Prepare          string
	StartBeforeScript bool
}

type Request struct {
	ScriptPath  string
	ServerPort  int
	TimeoutSecs int
	Headless    *bool
	ServiceSeed *ServiceSeed
}

type Response struct {
	ServerStarted  bool
	ServerPort     int
	ScriptExitCode int
	ScriptOutput   string
	ScriptResult   map[string]any
	BaseURL        string
	ConfigHome     string
	SeededService  map[string]any
	SeedError      string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	headless := true
	if req.Headless != nil {
		headless = *req.Headless
	}
	if req.ScriptPath == "" {
		req.ScriptPath = "script.js"
	}

	basePort := defaultQuickTestPort
	if req.ServerPort > 0 {
		basePort = req.ServerPort
	}
	hash := 0
	for _, c := range t.Name() {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	port := basePort + (hash % 100)
	resp.ServerPort = port

	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 120
	}

	projectRoot, err := findGoModuleRoot()
	if err != nil {
		return nil, fmt.Errorf("find repo root: %w", err)
	}
	if err := envpkg.Load(); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	caseDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get case dir: %w", err)
	}
	scriptPath := req.ScriptPath
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(caseDir, scriptPath)
	}
	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("read script %s: %w", scriptPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeoutSecs)*time.Second)
	defer cancel()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	resp.BaseURL = baseURL

	opts := lib.QuickTestOptions{
		Port:       port,
		ProjectDir: projectRoot,
		Local:      os.Getenv(lib.EnvQuickTestDefaultConfig) == lib.QuickTestDefaultConfigLocal,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}
	if err := lib.QuickTestPrepare(&opts); err != nil {
		return nil, fmt.Errorf("QuickTestPrepare: %w", err)
	}
	result, err := lib.QuickTestStart(ctx, &opts)
	if err != nil {
		return nil, fmt.Errorf("QuickTestStart: %w", err)
	}
	stopQuickTest := func() {
		if result != nil && result.ServerCmd != nil && result.ServerCmd.Process != nil {
			result.ServerCmd.Process.Signal(syscall.SIGTERM)
			_, _ = result.ServerCmd.Process.Wait()
		}
		if result != nil && result.ViteCmd != nil && result.ViteCmd.Process != nil {
			result.ViteCmd.Process.Signal(syscall.SIGTERM)
			_, _ = result.ViteCmd.Process.Wait()
		}
		lib.QuickTestCleanup(&opts)
	}
	t.Cleanup(stopQuickTest)

	if result != nil {
		resp.ConfigHome = result.ConfigHome
	}

	readyDeadline := time.Now().Add(time.Duration(req.TimeoutSecs) * time.Second)
	var started bool
	for time.Now().Before(readyDeadline) {
		if quickTestHealthy(baseURL) {
			started = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	resp.ServerStarted = started
	if !started {
		return resp, fmt.Errorf("server not ready on %s within %ds", baseURL, req.TimeoutSecs)
	}

	if req.ServiceSeed != nil {
		seeded, err := prepareServiceSeed(baseURL, req.ServiceSeed)
		if err != nil {
			resp.SeedError = err.Error()
		} else {
			resp.SeededService = seeded
		}
	}

	preamble := fmt.Sprintf("const BASE_URL = %q;\nconst CASE_DIR = %q;\n", baseURL, caseDir)
	fullScript := preamble + string(scriptBytes)

	headlessVal := headless
	req.Headless = &headlessVal

	var scriptOut bytes.Buffer
	scriptExitCode, scriptErr := runPlaywrightScript(ctx, headless, fullScript, &scriptOut, &scriptOut)
	resp.ScriptOutput = scriptOut.String()
	resp.ScriptExitCode = scriptExitCode
	resp.ScriptResult = parseLastJSONLine(resp.ScriptOutput)

	_ = scriptErr
	return resp, nil
}
```