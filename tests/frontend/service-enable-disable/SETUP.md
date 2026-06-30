# Scenario

**Feature**: frontend service enable/disable UI via Playwright + quick-test

```
# quick-test starts, API seeds one service, Playwright opens /home/service
Run -> POST /api/services (+ start/stop/disable) -> BASE_URL/home/service

# script clicks Disable or Enable and reads ConfirmModal message
leaf script.js -> modal prompt + status -> ScriptResult -> Assert
```

## Preconditions

1. Repository root contains `go.mod`; `playwright-debug` cache is available.
2. Quick-test serves the React app at `/home/service`.
3. Service disable/enable API endpoints and UI controls exist (tests fail until implemented).

## Steps

1. Root `Run` starts quick-test and waits for health.
2. `Request.ServiceSeed` drives API preparation (`running-enabled` or `stopped-disabled`).
3. Playwright script navigates to the service page and exercises Disable/Enable.
4. Leaf `Assert` checks `ScriptResult` and seeded API state.

## Context

Nested DOCTEST under `tests/frontend/` per REQUIREMENT-DESIGN-service-enable-disable.md.
Uses label `ui-automation` on runnable leaves.

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
	"testing"
	"time"
)

const defaultQuickTestPort = 3680

func Setup(t *testing.T, req *Request) error {
	if req.ScriptPath == "" {
		req.ScriptPath = "script.js"
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 120
	}
	return nil
}

func urlQueryEscape(id string) string {
	return strings.ReplaceAll(id, " ", "%20")
}

func postJSON(baseURL, path string, body map[string]any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func postServiceAction(baseURL, path, id string) error {
	req, err := http.NewRequest(http.MethodPost, baseURL+path+"?id="+urlQueryEscape(id), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func fetchServiceByID(baseURL, id string) (map[string]any, error) {
	resp, err := http.Get(baseURL + "/api/services")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /api/services status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var services []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, err
	}
	for _, svc := range services {
		if svcID, _ := svc["id"].(string); svcID == id {
			return svc, nil
		}
	}
	return nil, fmt.Errorf("service %q not found after seed", id)
}

func prepareServiceSeed(baseURL string, seed *ServiceSeed) (map[string]any, error) {
	id := seed.ID
	name := seed.Name
	command := seed.Command
	if id == "" {
		id = "ui-svc-001"
	}
	if name == "" {
		name = "ui-test-service"
	}
	if command == "" {
		command = "sleep 300"
	}

	body := map[string]any{
		"id":      id,
		"name":    name,
		"command": command,
	}
	if err := postJSON(baseURL, "/api/services", body); err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}

	switch seed.Prepare {
	case "running-enabled", "":
		if err := postServiceAction(baseURL, "/api/services/start", id); err != nil {
			return nil, fmt.Errorf("start service: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	case "stopped-disabled":
		if err := postServiceAction(baseURL, "/api/services/stop", id); err != nil {
			return nil, fmt.Errorf("stop service: %w", err)
		}
		if err := postServiceAction(baseURL, "/api/services/disable", id); err != nil {
			return nil, fmt.Errorf("disable service: %w", err)
		}
		time.Sleep(300 * time.Millisecond)
	default:
		return nil, fmt.Errorf("unknown prepare mode %q", seed.Prepare)
	}

	return fetchServiceByID(baseURL, id)
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
			return "", fmt.Errorf("go.mod not found")
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