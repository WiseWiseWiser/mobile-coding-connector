# Scenario

**Feature**: managed service enable/disable server + CLI harness

```
# build server (+ remote-agent), seed services.json, exercise API or CLI
leaf Setup -> services.json -> ai-critic-server -> disable/enable action -> status + message
```

## Preconditions

1. Module builds `ai-critic-server` (`.`) and `remote-agent` (`./cmd/remote-agent`).
2. Each test uses isolated `AI_CRITIC_HOME` with `lib.TestPassword` credentials.
3. Long-running service commands use `sleep` so PID checks remain stable.

## Steps

1. Root `Run` builds binaries, writes `services.json`, and starts the server.
2. Leaf `Setup` configures `Request.Services`, `Request.Action`, and pre-actions.
3. Root `Run` performs the disable/enable action (HTTP or CLI) and waits when needed.
4. Root `Run` snapshots `GET /api/services`, on-disk `services.json`, and target PID.
5. Leaf `Assert` verifies prompts, persistence, and process state.

## Context

Implements REQUIREMENT-DESIGN-service-enable-disable.md server and CLI leaves.
Assumes disable/enable do not synchronously stop or start processes; enable on a
stopped service schedules daemon reconcile within one 5s ticker window.

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

func boolPtr(v bool) *bool {
	return &v
}

func sleepService(id, name string, enabled *bool) ServiceSeed {
	return ServiceSeed{
		ID:      id,
		Name:    name,
		Command: "sleep 300",
		Enabled: enabled,
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

func enabledFieldOnDisk(rows []map[string]any, id string) (enabled *bool, present bool) {
	for _, row := range rows {
		rowID, _ := row["id"].(string)
		if rowID != id {
			continue
		}
		if v, ok := row["enabled"].(bool); ok {
			enabled = &v
			present = true
			return enabled, present
		}
		return nil, false
	}
	return nil, false
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

func serviceNameForTarget(req *Request) string {
	for _, seed := range req.Services {
		if seed.ID == req.TargetID {
			return seed.Name
		}
	}
	return req.TargetID
}

func urlQueryEscape(id string) string {
	return strings.ReplaceAll(id, " ", "%20")
}

func writeServicesJSON(configHome string, services []ServiceSeed) error {
	now := "2026-06-30T00:00:00Z"
	rows := make([]servicesFileRow, 0, len(services))
	for _, svc := range services {
		if svc.ID == "" || svc.Name == "" || svc.Command == "" {
			return fmt.Errorf("service id, name, and command are required")
		}
		rows = append(rows, servicesFileRow{
			ID:        svc.ID,
			Name:      svc.Name,
			Command:   svc.Command,
			Enabled:   svc.Enabled,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configHome, "services.json"), data, 0644)
}

func readServicesJSON(configHome string) ([]map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(configHome, "services.json"))
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	return rows, nil
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

func postServiceAction(baseURL, token, path, id string) (*serviceActionResponse, error) {
	url := baseURL + path + "?id=" + urlQueryEscape(id)
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
		return nil, fmt.Errorf("%s status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out serviceActionResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode %s: %w body=%s", path, err, strings.TrimSpace(string(body)))
	}
	return &out, nil
}

func writeRemoteAgentConfig(path, server, token string) error {
	cfg := map[string]any{
		"default": server,
		"domains": []map[string]string{{"server": server, "token": token}},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func stripEnvPrefix(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
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
	return 26000 + (hash % 1000)
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