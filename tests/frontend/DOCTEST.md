# Frontend Doctests

Doc-style tests that drive the React UI through Playwright scripts executed via
the `playwright-debug` CLI. Each leaf keeps its browser automation in a
`script.js` fixture alongside `SETUP.md`.

# DSN (Domain Specific Notion)

The frontend doctest harness models a browser-driven smoke-test system for the
ai-critic React app.

**Participants**

- **Doctest runner** — walks the decision tree, chains `Setup` functions,
  calls root `Run`, then leaf `Assert`.
- **Quick-test server** — builds and starts the Go backend on an isolated temp
  `AI_CRITIC_HOME` with test credentials; proxies to Vite.
- **Vite dev server** — serves the React frontend for browser navigation.
- **Playwright** — headless Chromium executes each leaf `script.js`, printing a
  JSON result line to stdout.
- **File Transfer store** — flat `{AI_CRITIC_HOME}/file-transfer/` directory
  backing the dedicated `/api/file-transfer` endpoints and `FileTransferView`.
- **FileTransferView** — WeChat-like inbox: list uploaded files, upload
  (button + drag-and-drop), download, and remove (with confirmation).

**Behaviors**

- Root `Run` starts quick-test, waits for health, optionally seeds the file
  transfer directory, injects `BASE_URL` and `CASE_DIR` into the script, runs
  Playwright, and parses `ScriptResult`.
- Navigation leaves verify routes render expected headings and shell UI.
- File-transfer leaves verify list state, upload, download, and delete against
  the dedicated API and UI.

## Version

0.0.2

## Decision Tree

```
[frontend tests]
 |
 +-- navigation/                          (grouping — route smoke tests)
 |    |
 |    +-- home-loads/                     (LEAF)  /home — workspace list
 |    +-- tools-loads/                    (LEAF)  /home/tools — Server Tools
 |    +-- settings-loads/                 (LEAF)  /home/settings — Settings
 |    +-- root-redirects-home/            (LEAF)  / → /home redirect
 |    +-- file-transfer-loads/            (LEAF)  /home/file-transfer — page shell
 |
 +-- setup/                               (grouping — first-launch Setup page)
 |    |
 |    +-- generate-random-uninitialized/ (LEAF)  Generate Random fills credential
 |
 +-- file-transfer/                        (grouping — inbox operations)
      |
      +-- list-empty/                     (LEAF)  empty dir → empty-state message
      +-- upload-and-list/                (LEAF)  UI upload → row appears
      +-- download-file/                  (LEAF)  seeded file → browser download
      +-- delete-file/                    (LEAF)  seeded file → remove + API gone
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `navigation/home-loads` | Navigates to `/home` and verifies the workspace list UI renders |
| 2 | `navigation/tools-loads` | Navigates to `/home/tools` and verifies the Server Tools page loads |
| 3 | `navigation/settings-loads` | Navigates to `/home/settings` and verifies the Settings heading is visible |
| 4 | `navigation/root-redirects-home` | Navigates to `/` and verifies redirect to `/home` |
| 5 | `navigation/file-transfer-loads` | Navigates to `/home/file-transfer`; heading and upload area visible |
| 6 | `setup/generate-random-uninitialized` | Uninitialized server: Generate Random fills 64-char credential (no error) |
| 7 | `file-transfer/list-empty` | Empty `file-transfer/` dir shows empty-state message; file count 0 |
| 8 | `file-transfer/upload-and-list` | Upload `testdata/sample.txt` via UI; row with name and size appears |
| 9 | `file-transfer/download-file` | Pre-seeded `hello.txt`; Download triggers save as `hello.txt` |
| 10 | `file-transfer/delete-file` | Pre-seeded `temp.txt`; Remove confirms; row gone and absent from API |

## Parameter Coverage

| Leaf | Route | Storage state | Operation | TimeoutSecs |
|------|-------|---------------|-----------|-------------|
| home-loads | `/home` | — | page load | 90 |
| tools-loads | `/home/tools` | — | page load | 120 |
| settings-loads | `/home/settings` | — | page load | 90 |
| root-redirects-home | `/` | — | redirect | 90 |
| file-transfer-loads | `/home/file-transfer` | any | page load | 90 |
| generate-random-uninitialized | `/` (Setup) | uninitialized | generate | 120 |
| list-empty | `/home/file-transfer` | empty (reset) | list | 90 |
| upload-and-list | `/home/file-transfer` | empty (reset) | upload + list | 120 |
| download-file | `/home/file-transfer` | seeded `hello.txt` | download | 90 |
| delete-file | `/home/file-transfer` | seeded `temp.txt` | delete | 90 |

## Harness Behaviour (root `Run`)

1. Resolves repo root and starts quick-test server (`lib.QuickTestPrepare` + `lib.QuickTestStart`)
2. Waits for `/api/quick-test/health` or `/ping`
3. Optionally resets/seeds `{AI_CRITIC_HOME}/file-transfer/` per `Request.FileTransferReset` and `Request.FileTransferSeeds`
4. Reads leaf `script.js` from the case directory (working directory at runtime)
5. Injects `const BASE_URL` and `const CASE_DIR` preamble
6. Executes via headless Playwright (default) or `playwright-debug run` when visible
7. Parses the last JSON object line from stdout into `Response.ScriptResult`
8. Tears down quick-test server and Vite on cleanup

## How to Run

Validate tree structure:

```sh
doctest vet ./tests/frontend
```

Run all frontend doctests:

```sh
doctest test ./tests/frontend/...
```

Run navigation leaves:

```sh
doctest test ./tests/frontend/navigation/home-loads
doctest test ./tests/frontend/navigation/file-transfer-loads
```

Run file-transfer leaves:

```sh
doctest test ./tests/frontend/file-transfer/list-empty
doctest test ./tests/frontend/file-transfer/upload-and-list
doctest test ./tests/frontend/file-transfer/download-file
doctest test ./tests/frontend/file-transfer/delete-file
```

Post-implementation verification:

```sh
go run ./script/build
doctest test ./tests/frontend/...
doctest test ./...
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

const defaultQuickTestPort = 3580

type FileTransferSeed struct {
	Name       string
	SourcePath string
}

type Request struct {
	ScriptPath        string
	ServerPort        int
	TimeoutSecs       int
	Headless          *bool
	Uninitialized     bool
	FileTransferReset bool
	FileTransferSeeds []FileTransferSeed
}

type Response struct {
	ServerStarted  bool
	ServerPort     int
	ScriptExitCode int
	ScriptOutput   string
	ScriptResult   map[string]any
	BaseURL        string
	ConfigHome     string
}

type serverProcess struct {
	cmd        *exec.Cmd
	viteCmd    *exec.Cmd
	configHome string
	binPath    string
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
		req.TimeoutSecs = 90
	}

	projectRoot, err := findGoModuleRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find repo root: %w", err)
	}

	if err := envpkg.Load(); err != nil {
		return nil, fmt.Errorf("failed to load env: %w", err)
	}

	caseDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get case directory: %w", err)
	}
	scriptPath := req.ScriptPath
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(caseDir, scriptPath)
	}

	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read playwright fixture %s: %w", scriptPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeoutSecs)*time.Second)
	defer cancel()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	resp.BaseURL = baseURL

	var configHome string
	if req.Uninitialized {
		proc, err := startUninitializedServer(ctx, t, projectRoot, port)
		if err != nil {
			return nil, fmt.Errorf("start uninitialized server: %w", err)
		}
		configHome = proc.configHome
		resp.ConfigHome = configHome
	} else {
		opts := lib.QuickTestOptions{
			Port:       port,
			ProjectDir: projectRoot,
			Local:      os.Getenv(lib.EnvQuickTestDefaultConfig) == lib.QuickTestDefaultConfigLocal,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
		}

		if err := lib.QuickTestPrepare(&opts); err != nil {
			return nil, fmt.Errorf("QuickTestPrepare failed: %w", err)
		}

		result, err := lib.QuickTestStart(ctx, &opts)
		if err != nil {
			return nil, fmt.Errorf("QuickTestStart failed: %w", err)
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
			configHome = result.ConfigHome
			resp.ConfigHome = configHome
		}
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
	if err := prepareFileTransfer(req, resp.ConfigHome, caseDir); err != nil {
		return resp, fmt.Errorf("prepare file-transfer dir: %w", err)
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

	if scriptErr != nil {
		return resp, fmt.Errorf("playwright script failed (exit %d): %w\n%s", resp.ScriptExitCode, scriptErr, resp.ScriptOutput)
	}

	return resp, nil
}

func findGoModuleRoot() (string, error) {
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
			return "", fmt.Errorf("go.mod not found in any parent of %s", dir)
		}
		dir = parent
	}
}

func quickTestHealthy(baseURL string) bool {
	for _, path := range []string{"/api/quick-test/health", "/ping"} {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(baseURL + path)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true
		}
	}
	return false
}

func playwrightCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".playwright-debug", "node_package")
	}
	return filepath.Join(home, ".playwright-debug", "node_package")
}

func exitCodeFromCmd(cmd *exec.Cmd, err error) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	if err != nil {
		return 1
	}
	return 0
}

func runHeadlessPlaywrightScript(ctx context.Context, script string, stdout, stderr io.Writer) (int, error) {
	dir := playwrightCacheDir()
	wrapper := fmt.Sprintf(`
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ channel: 'chromium-headless-shell', headless: true });
  const page = await browser.newPage();
  try {
    %s
  } finally {
    await browser.close();
  }
})();
`, script)

	cmd := exec.CommandContext(ctx, "node", "-e", wrapper)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "CI=true")
	err := cmd.Run()
	return exitCodeFromCmd(cmd, err), err
}

func runPlaywrightScript(ctx context.Context, headless bool, script string, stdout, stderr io.Writer) (int, error) {
	if headless {
		return runHeadlessPlaywrightScript(ctx, script, stdout, stderr)
	}

	cmd := exec.CommandContext(ctx, "playwright-debug", "run", script)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "CI=true")
	err := cmd.Run()
	return exitCodeFromCmd(cmd, err), err
}

func fileTransferDir(configHome string) string {
	return filepath.Join(configHome, "file-transfer")
}

func prepareFileTransfer(req *Request, configHome, caseDir string) error {
	if configHome == "" {
		return nil
	}
	if !req.FileTransferReset && len(req.FileTransferSeeds) == 0 {
		return nil
	}

	dir := fileTransferDir(configHome)
	if req.FileTransferReset {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("reset file-transfer dir: %w", err)
		}
	}
	if req.FileTransferReset || len(req.FileTransferSeeds) > 0 {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create file-transfer dir: %w", err)
		}
	}
	for _, seed := range req.FileTransferSeeds {
		src := seed.SourcePath
		if !filepath.IsAbs(src) {
			src = filepath.Join(caseDir, src)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read seed %q: %w", seed.SourcePath, err)
		}
		dst := filepath.Join(dir, seed.Name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write seed %q: %w", seed.Name, err)
		}
	}
	return nil
}

func fetchFileTransferNames(baseURL string) ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/api/file-transfer")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /api/file-transfer status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Files []struct {
			Name string `json:"name"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(payload.Files))
	for _, f := range payload.Files {
		names = append(names, f.Name)
	}
	return names, nil
}

func parseLastJSONLine(output string) map[string]any {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var result map[string]any
		if err := json.Unmarshal([]byte(line), &result); err == nil {
			return result
		}
	}
	return nil
}

func startUninitializedServer(ctx context.Context, t *testing.T, projectRoot string, port int) (*serverProcess, error) {
	opts := lib.QuickTestOptions{
		Port:       port,
		ProjectDir: projectRoot,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}
	if err := lib.QuickTestPrepare(&opts); err != nil {
		return nil, fmt.Errorf("QuickTestPrepare: %w", err)
	}
	binPath := "/tmp/ai-critic-quick"

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })

	credFile := lib.TestCredentialsFile(configHome)
	frontendPort := lib.ViteDevPort

	var viteCmd *exec.Cmd
	viteURL := fmt.Sprintf("http://localhost:%d", frontendPort)
	viteUp := func() bool {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(viteURL)
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}
	if !viteUp() {
		viteCmd = exec.CommandContext(ctx, "npm", "run", "dev")
		viteCmd.Dir = filepath.Join(projectRoot, "ai-critic-react")
		viteCmd.Stdout = io.Discard
		viteCmd.Stderr = io.Discard
		if err := viteCmd.Start(); err != nil {
			return nil, fmt.Errorf("start vite: %w", err)
		}
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			if viteUp() {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !viteUp() {
			if viteCmd.Process != nil {
				viteCmd.Process.Kill()
			}
			return nil, fmt.Errorf("vite not ready on %s", viteURL)
		}
	}

	args := []string{
		"--port", strconv.Itoa(port),
		"--credentials-file", credFile,
		"--project-dir", projectRoot,
		"--dev",
		"--frontend-port", strconv.Itoa(frontendPort),
	}
	serverCmd := exec.Command(binPath, args...)
	serverCmd.Dir = configHome
	serverCmd.Stdout = io.Discard
	serverCmd.Stderr = io.Discard
	serverCmd.Env = lib.AppendTestServerEnv(os.Environ(), configHome)

	if err := serverCmd.Start(); err != nil {
		if viteCmd != nil && viteCmd.Process != nil {
			viteCmd.Process.Kill()
		}
		return nil, fmt.Errorf("start server: %w", err)
	}

	stop := func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			_, _ = serverCmd.Process.Wait()
		}
		if viteCmd != nil && viteCmd.Process != nil {
			viteCmd.Process.Signal(syscall.SIGTERM)
			_, _ = viteCmd.Process.Wait()
		}
	}
	t.Cleanup(stop)

	return &serverProcess{
		cmd:        serverCmd,
		viteCmd:    viteCmd,
		configHome: configHome,
		binPath:    binPath,
	}, nil
}
```