# Scenario

**Feature**: cron task management server + remote-agent CLI harness

```
# build session-cached server + remote-agent, isolate AI_CRITIC_HOME, exercise API or CLI
leaf Setup -> cron-tasks.json seeds -> ai-critic-server tick loop
  -> HTTP /api/cron-tasks* | remote-agent cron …
  -> list / history / marker evidence
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/cron-task-management-doctest-<session>/` (binaries built once per run).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. Each leaf gets isolated `AI_CRITIC_HOME` (`lib.CreateTestConfigHome`), work dir for
   marker files, and `agentHome` for remote-agent config. Only compiled binaries are shared.
4. Module builds `ai-critic-server` (`.`) and `remote-agent` (`./cmd/remote-agent`).
5. Test credentials use `lib.TestPassword`.

## Steps

1. Root `Run` builds/reuses session binaries, seeds optional `cron-tasks.json`, starts server.
2. Leaf `Setup` sets `Request` fields (action, schedule, waits, CLI args).
3. Root `Run` performs API or CLI action, optional wait/poll, snapshots list/history/marker.
4. Leaf `Assert` verifies status, history, files, CLI stdout/exit.

## Context

Implements REQUIREMENT-DESIGN-cron-task-management.md. Locked rules: interval
finish-based, overlap skip only, default timeout 1h always enforced, server UTC,
CLI `--cron` local convert with unsafe error → `--cron-utc`, 7d history, global only.

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

func sessionCacheDir() string {
	return filepath.Join(os.TempDir(), "cron-task-management-doctest-"+DOCTEST_SESSION_ID)
}

func withFileLock(t *testing.T, lockPath string, fn func() error) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock %s: %w", lockPath, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func buildSessionBinariesOnce(t *testing.T, moduleRoot, cacheDir string) (serverBin, agentBin string) {
	t.Helper()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	serverBin = filepath.Join(cacheDir, "ai-critic-server")
	agentBin = filepath.Join(cacheDir, "remote-agent")
	ready := filepath.Join(cacheDir, "binaries.ready")
	lock := filepath.Join(cacheDir, "build.lock")
	err := withFileLock(t, lock, func() error {
		if fileExists(ready) && fileExists(serverBin) && fileExists(agentBin) {
			return nil
		}
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
				return fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
			}
		}
		return os.WriteFile(ready, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("session binaries cache: %s", cacheDir)
	return serverBin, agentBin
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func boolPtr(v bool) *bool {
	return &v
}

func findTaskByName(tasks []cronTaskStatus, name string) (cronTaskStatus, bool) {
	for _, t := range tasks {
		if t.Name == name {
			return t, true
		}
	}
	return cronTaskStatus{}, false
}

func findTaskByID(tasks []cronTaskStatus, id string) (cronTaskStatus, bool) {
	for _, t := range tasks {
		if t.ID == id {
			return t, true
		}
	}
	return cronTaskStatus{}, false
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func writeCronTasksJSON(configHome string, tasks []TaskSeed) error {
	now := "2026-07-10T00:00:00Z"
	rows := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		if task.ID == "" || task.Name == "" || task.Command == "" {
			return fmt.Errorf("task id, name, and command are required")
		}
		if task.ScheduleMode == "" {
			return fmt.Errorf("task %s: scheduleMode required", task.ID)
		}
		row := map[string]any{
			"id":           task.ID,
			"name":         task.Name,
			"command":      task.Command,
			"scheduleMode": task.ScheduleMode,
			"createdAt":    now,
			"updatedAt":    now,
		}
		if task.WorkingDir != "" {
			row["workingDir"] = task.WorkingDir
		}
		if task.Interval != "" {
			row["interval"] = task.Interval
		}
		if task.CronExpr != "" {
			row["cronExpr"] = task.CronExpr
		}
		if task.Timeout != "" {
			row["timeout"] = task.Timeout
		}
		if task.Enabled != nil {
			row["enabled"] = *task.Enabled
		}
		if len(task.RecentRuns) > 0 {
			row["recentRuns"] = task.RecentRuns
		}
		rows = append(rows, row)
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configHome, "cron-tasks.json"), data, 0644)
}

func getCronTasks(baseURL, token string) ([]cronTaskStatus, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/cron-tasks", nil)
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /api/cron-tasks status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	trim := strings.TrimSpace(string(body))
	if trim == "" {
		return nil, fmt.Errorf("GET /api/cron-tasks empty body")
	}
	if trim[0] != '[' && trim[0] != '{' {
		return nil, fmt.Errorf("GET /api/cron-tasks non-JSON body (cron API missing?): %s", truncate(trim, 120))
	}
	var out []cronTaskStatus
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode cron-tasks list: %w body=%s", err, truncate(trim, 200))
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func getCronHistory(baseURL, token, id string) ([]cronTaskRun, error) {
	url := baseURL + "/api/cron-tasks/history?id=" + urlQueryEscape(id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET history status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out []cronTaskRun
	if err := json.Unmarshal(body, &out); err != nil {
		// allow object wrapper {"runs":[...]}
		var wrap struct {
			Runs []cronTaskRun `json:"runs"`
		}
		if err2 := json.Unmarshal(body, &wrap); err2 != nil {
			return nil, fmt.Errorf("decode history: %w body=%s", err, strings.TrimSpace(string(body)))
		}
		return wrap.Runs, nil
	}
	return out, nil
}

func postCronCreate(baseURL, token string, body map[string]any) (int, string, *cronTaskStatus, error) {
	return doCronJSON(http.MethodPost, baseURL+"/api/cron-tasks", token, body)
}

func putCronUpdate(baseURL, token string, body map[string]any) (int, string, *cronTaskStatus, error) {
	return doCronJSON(http.MethodPut, baseURL+"/api/cron-tasks", token, body)
}

func doCronJSON(method, url, token string, body map[string]any) (int, string, *cronTaskStatus, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return 0, "", nil, err
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	rawStr := string(raw)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Prefer short body for HTML 404 pages
		msg := strings.TrimSpace(rawStr)
		if strings.HasPrefix(msg, "<!") || strings.HasPrefix(msg, "<html") {
			msg = fmt.Sprintf("(html response, cron API likely missing) status=%d", resp.StatusCode)
		}
		return resp.StatusCode, rawStr, nil, fmt.Errorf("%s %s status %d: %s", method, url, resp.StatusCode, truncate(msg, 200))
	}
	var st cronTaskStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		// allow wrapper {"task":{...}} or {"status":...}
		var wrap struct {
			Task cronTaskStatus `json:"task"`
		}
		if err2 := json.Unmarshal(raw, &wrap); err2 != nil || wrap.Task.ID == "" {
			return resp.StatusCode, rawStr, nil, fmt.Errorf("decode create/update: %w body=%s", err, strings.TrimSpace(rawStr))
		}
		st = wrap.Task
	}
	return resp.StatusCode, rawStr, &st, nil
}

func postCronAction(baseURL, token, path, id string) (int, string, error) {
	url := baseURL + path + "?id=" + urlQueryEscape(id)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, string(raw), fmt.Errorf("%s status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return resp.StatusCode, string(raw), nil
}

func deleteCronTask(baseURL, token, id string) (int, string, error) {
	url := baseURL + "/api/cron-tasks?id=" + urlQueryEscape(id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, string(raw), fmt.Errorf("DELETE status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return resp.StatusCode, string(raw), nil
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

func urlQueryEscape(id string) string {
	return strings.ReplaceAll(id, " ", "%20")
}

func findModuleRoot() (string, error) {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		for dir := root; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
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

func parseRFC3339(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	return time.Parse(time.RFC3339, s)
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}
```
