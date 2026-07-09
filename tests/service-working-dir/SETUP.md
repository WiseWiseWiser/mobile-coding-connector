# Scenario

**Feature**: managed service workingDir auto-create on start

```
# seed services.json with workingDir, start via API
leaf Setup -> services.json(workingDir) -> ai-critic-server -> POST /api/services/start

# verify disk, status, and service log
GET /api/services + Stat(workingDir) + read services/{id}.log
```

## Preconditions

1. Module builds `ai-critic-server` (`.`).
2. Each test uses isolated `AI_CRITIC_HOME` with `lib.TestPassword` credentials.
3. Long-running service commands use `sleep 300` so PID checks remain stable.
4. Missing-dir leaves do **not** pre-create `workingDir`; existing-dir leaf does.

## Steps

1. Root `Run` builds the server binary, writes `services.json`, and starts the server.
2. Leaf `Setup` configures `Request.Services`, `Request.WorkingDir`, and target id.
3. Root `Run` calls `POST /api/services/start` and snapshots status, disk, and log.
4. Leaf `Assert` verifies directory creation, running PID, and log contents.

## Context

Implements REQUIREMENT-DESIGN-service-working-dir.md. Exercises
`ensureServiceWorkingDir` → `os.MkdirAll` before `exec.Command` in
`server/services`.

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

func workingDirService(id, name, workingDir string) ServiceSeed {
	return ServiceSeed{
		ID:         id,
		Name:       name,
		Command:    "sleep 300",
		WorkingDir: workingDir,
	}
}

func findServiceByID(services []serviceStatus, id string) (serviceStatus, bool) {
	for _, svc := range services {
		if svc.ID == id {
			return svc, true
		}
	}
	return serviceStatus{}, false
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func serviceIsRunning(svc serviceStatus) bool {
	if svc.PID > 0 {
		return processAlive(svc.PID)
	}
	return svc.Status == "running" || svc.Status == "starting"
}

func urlQueryEscape(id string) string {
	return strings.ReplaceAll(id, " ", "%20")
}

func writeServicesJSON(configHome string, services []ServiceSeed) error {
	now := "2026-07-09T00:00:00Z"
	rows := make([]servicesFileRow, 0, len(services))
	for _, svc := range services {
		if svc.ID == "" || svc.Name == "" || svc.Command == "" {
			return fmt.Errorf("service id, name, and command are required")
		}
		rows = append(rows, servicesFileRow{
			ID:         svc.ID,
			Name:       svc.Name,
			Command:    svc.Command,
			WorkingDir: svc.WorkingDir,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configHome, "services.json"), data, 0644)
}

func getServices(baseURL, token string) ([]serviceStatus, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/services", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /api/services status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out []serviceStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func postServiceStart(baseURL, token, id string) (*serviceStatus, error) {
	url := baseURL + "/api/services/start?id=" + urlQueryEscape(id)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("POST /api/services/start status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out serviceStatus
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode start response: %w body=%s", err, strings.TrimSpace(string(body)))
	}
	return &out, nil
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

func portBaseFromTestName(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return 27000 + (hash % 1000)
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