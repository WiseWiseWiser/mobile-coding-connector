# Scenario

**Feature**: frontend smoke and file-transfer tests via Playwright + quick-test

```
# doctest starts isolated quick-test + Vite, then runs Playwright script
doctest Run -> quick-test server + Vite -> BASE_URL

# leaf script.js drives browser; JSON line becomes ScriptResult for Assert
leaf script.js -> Playwright -> ScriptResult -> Assert

# file-transfer leaves may reset/seed {AI_CRITIC_HOME}/file-transfer/ before script
Run -> file-transfer dir (reset/seed) -> FileTransferView + /api/file-transfer
```

## Preconditions

1. The repository root is discoverable (contains `go.mod`).
2. `playwright-debug` is available on `PATH`.
3. Node/npm are available for the Vite dev server started by quick-test.
4. Each test runs with working directory set to its leaf case directory.

## Steps

1. Resolve the repository root and compute a per-test server port (default `3580` plus a hash offset from the test name).
2. Call `lib.QuickTestPrepare` and `lib.QuickTestStart` to build and start the quick-test server and Vite dev server. Unless `Local` mode is enabled, quick-test uses an isolated temp `AI_CRITIC_HOME` with `testpassword` in `server-credentials` (auth is bypassed in quick-test mode).
3. Wait until `/api/quick-test/health` or `/ping` responds on the chosen port.
4. Read the leaf Playwright fixture from `Request.ScriptPath` (default `script.js`, relative to the case directory).
5. Prepend `const BASE_URL = "http://localhost:<port>";` to the script body.
6. Execute the script headlessly by default (`Request.Headless` defaults to `true`):
   - When `Headless=true`: run via `node` using the `playwright-debug` cache
     (`~/.playwright-debug/node_package`) with the Chromium headless-shell channel
     (`chromium.launch({ channel: 'chromium-headless-shell', headless: true })`).
   - When `Headless=false`: run via `playwright-debug run` for visible debugging.
7. Set `CI=true` on the script process environment.
8. Capture stdout/stderr and exit code; parse the last JSON object line into `Response.ScriptResult`.
9. Tear down the quick-test server and Vite processes on cleanup.

## Context

These tests verify frontend navigation smoke scenarios by driving a real browser
against a quick-test server instance. Each leaf supplies a `script.js` fixture
that prints a single JSON line to stdout for machine-readable assertions.

### Parameters (ranked by significance)

| # | Parameter | Type | Values | Description |
|---|-----------|------|--------|-------------|
| 1 | Route (leaf) | path | `/home`, `/home/tools`, `/home/settings`, `/` | Which page or redirect behaviour is exercised (encoded in `script.js`) |
| 2 | `ScriptPath` | string | `script.js` (default) | Playwright fixture filename relative to leaf directory |
| 3 | `ServerPort` | int | 0 → 3580 + hash offset | Quick-test server listen port |
| 4 | `TimeoutSecs` | int | 90–120 | Server readiness and script execution budget |
| 5 | `Headless` | bool | true (default) | Headless shell mode via Playwright; set `false` in leaf `Setup` for visible debugging |
| 6 | `FileTransferReset` | bool | true/false | When true, remove and recreate empty `{AI_CRITIC_HOME}/file-transfer/` before the script |
| 7 | `FileTransferSeeds` | []FileTransferSeed | name + source path | Files copied into `file-transfer/` after server is healthy |

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
	ScriptPath         string
	ServerPort         int
	TimeoutSecs        int
	Headless           *bool // nil/true → headless shell; false → visible playwright-debug (opt-in)
	FileTransferReset  bool
	FileTransferSeeds  []FileTransferSeed
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

	opts := lib.QuickTestOptions{
		Port:        port,
		ProjectDir:  projectRoot,
		Local:       os.Getenv(lib.EnvQuickTestDefaultConfig) == lib.QuickTestDefaultConfigLocal,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
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

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	resp.BaseURL = baseURL

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
		return resp, fmt.Errorf("quick-test server not ready on %s within %ds", baseURL, req.TimeoutSecs)
	}

	if result != nil {
		resp.ConfigHome = result.ConfigHome
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
```