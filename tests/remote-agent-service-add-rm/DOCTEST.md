# Remote-Agent Service Add + Rm Doctests

Classic TDD doctests for `remote-agent service add` and `service rm`, plus
`service list --all` and name resolution across `projectDir` scopes (ListAll).

# DSN (Domain Specific Notion)

**L2 only** (unlabeled mass). No product binary e2e.

In-process harness: `services.NewManagerAt` + `RegisterAPIWithManager` +
`agentcli.Run` against an httptest mux with bearer auth (`lib.TestPassword`).
Stdout/stderr capture is serialized with a package mutex (agentcli swaps
globals). No `t.Setenv` / `t.Chdir` for product correctness; config isolation
via `lib.CreateTestConfigHome` and temp dirs.

**Participants**

- **L2: services.Manager** — isolated `configHome/services.json` via
  `NewManagerAt`; `ListAll` / disk snapshot after CLI.
- **L2: agentcli.Run** — in-process `service add|rm|list` against local mux
  (`--server` + `--token`).
- **HTTP API (existing server)** — `POST /api/services`, `DELETE /api/services?id=`,
  `GET /api/services?all=1` (already implemented; CLI/client wiring is greenfield).
- **Service definitions** — name, command, optional `projectDir` / `workingDir` /
  `enabled`; long-running `sleep` for `--start` PID checks.

**Behaviors**

- **`service add`** — POST create; default definition-only (no start) unless
  `--start`; prints Created + service summary; optional `--disabled`.
- **`service rm <name-or-id>`** — resolve via list-all, DELETE by id; prints Removed.
- **`service list --all`** — cross-project listing via `?all=1`.
- **`resolveServiceTarget`** — name/id resolution uses list-all so services under
  any `projectDir` are visible to `rm` (and start/stop/update later).
- **Errors** — missing `--name`/`--command`; unknown target; ambiguous name.
- **No aliases** — only `add` and `rm` (no create/delete/remove).

## Version

0.0.2

## Decision Tree

```
[remote-agent service add + rm]
 |
 +-- add/                               (GROUP)  service add
 |    +-- happy-path/                   (LEAF)  name+command+dirs → Created; on disk; list-all sees it
 |    +-- missing-name/                 (LEAF)  no --name → non-zero
 |    +-- missing-command/              (LEAF)  no --command → non-zero
 |    +-- with-start/                   (LEAF)  --start → PID>0 / running
 |    +-- with-disabled/                (LEAF)  --disabled → enabled=false on disk
 |
 +-- rm/                                (GROUP)  service rm
 |    +-- by-name/                      (LEAF)  seed then rm by name → gone; Removed
 |    +-- by-id/                        (LEAF)  rm by id → gone; Removed
 |    +-- not-found/                    (LEAF)  unknown target → non-zero
 |    +-- cross-scope/                  (LEAF)  non-default projectDir; rm by name via ListAll
 |
 +-- list/                              (GROUP)  service list
      +-- all/                          (LEAF)  two projectDirs; list --all shows both
      +-- scoped/                       (LEAF)  list --project-dir LOCAL hides other
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `add/happy-path` | `--name` + `--command` + dirs → exit 0, Created, services.json row, list-all |
| 2 | `add/missing-name` | omit `--name` → non-zero error |
| 3 | `add/missing-command` | omit `--command` → non-zero error |
| 4 | `add/with-start` | `--start` with `sleep 30` → running / PID > 0 |
| 5 | `add/with-disabled` | `--disabled` → `enabled: false` on disk |
| 6 | `rm/by-name` | seed + `rm <name>` → Removed; gone from ListAll |
| 7 | `rm/by-id` | seed + `rm <id>` → Removed; gone from ListAll |
| 8 | `rm/not-found` | `rm missing` → non-zero |
| 9 | `rm/cross-scope` | seed under other projectDir; `rm <name>` resolves via all |
| 10 | `list/all` | two projectDirs; `list --all` shows both names |
| 11 | `list/scoped` | `list --project-dir LOCAL` shows only local name |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Subcommand: add | add/* |
| Subcommand: rm | rm/* |
| Subcommand: list | list/* |
| Required flags present/absent | add/happy-path, missing-name, missing-command |
| Start after add | add/with-start |
| Enabled flag | add/with-disabled |
| Resolve by name | rm/by-name, rm/cross-scope |
| Resolve by id | rm/by-id |
| Resolve miss | rm/not-found |
| Cross-project visibility (ListAll) | rm/cross-scope, list/all |
| Scoped list (project-dir) | list/scoped |

## Locked product API (implementer)

Classic TDD: CLI surface and client helpers below are **not** required to exist
yet. Harness compiles against current `agentcli` + `services` packages; leaves
are **RED** until implementer lands wiring.

| Surface | Role |
|---------|------|
| `client.ListAllServices()` | `GET /api/services?all=1` |
| `client.DeleteService(id)` | `DELETE /api/services?id=` |
| `agentcli` `service add` | flags mirror update; POST SaveService; optional `--start` / `--disabled` |
| `agentcli` `service rm` | resolve + DeleteService; print Removed |
| `agentcli` `service list --all` | ListAllServices |
| `resolveServiceTarget` | use ListAll (not scoped ListServices("")) |

Server Manager POST/DELETE/`?all=1` already exist — do not re-implement server.

## How to Run

```sh
cd /Users/xhd2015/Projects/xhd2015/lifelog/external/ai-critic-master-2026-07-30
doctest vet ./tests/remote-agent-service-add-rm
doctest test ./tests/remote-agent-service-add-rm
# or:
doctest test ./tests/remote-agent-service-add-rm/...
```

Classic TDD: expect **RED** (unknown subcommand / missing client APIs) until implementer lands CLI + client.

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// agentcliInProcessMu serializes in-process agentcli.Run (stdout/stderr swaps).
var agentcliInProcessMu sync.Mutex

// ServiceSeed is one row written to services.json before the L2 Manager starts.
type ServiceSeed struct {
	ID         string
	Name       string
	Command    string
	ProjectDir string
	WorkingDir string
	Enabled    *bool
}

// Request configures one leaf. All leaves are L2 CLI via agentcli.Run.
type Request struct {
	// CLIArgs is argv after global flags (include leading "service").
	CLIArgs []string

	// Services seeds services.json before Manager load (rm/list leaves).
	Services []ServiceSeed

	// TargetID / TargetName help Assert + post-CLI snapshots.
	TargetID   string
	TargetName string

	// LocalProjectDir / OtherProjectDir are absolute paths for multi-scope leaves.
	LocalProjectDir string
	OtherProjectDir string

	Token         string
	WaitAfterSecs int // settle after --start
}

type serviceStatus struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Command        string `json:"command"`
	ProjectDir     string `json:"projectDir,omitempty"`
	WorkingDir     string `json:"workingDir,omitempty"`
	Status         string `json:"status"`
	PID            int    `json:"pid"`
	DesiredRunning bool   `json:"desiredRunning"`
	Enabled        bool   `json:"enabled"`
}

type servicesFileRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	ProjectDir string `json:"projectDir,omitempty"`
	WorkingDir string `json:"workingDir,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ConfigHome string
	AgentHome  string
	ServerURL  string

	ServicesAfter  []serviceStatus
	ServicesOnDisk []map[string]any
	TargetID       string
	TargetName     string
	TargetPID      int
	TargetRunning  bool
	TargetEnabled  *bool
	ListedNames    []string
	ListedIDs      []string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if len(req.CLIArgs) == 0 {
		return nil, fmt.Errorf("CLIArgs required")
	}

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	agentHome, err := os.MkdirTemp("", "remote-agent-service-add-rm-home-*")
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

	m := services.NewManagerAt(configHome)
	t.Cleanup(func() { m.Shutdown() })

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
	resp.ServerURL = srv.URL

	if err := writeRemoteAgentConfig(filepath.Join(aiCriticAgent, "remote-agent-config.json"), srv.URL, req.Token); err != nil {
		return nil, err
	}

	fullArgv := append([]string{"--server", srv.URL, "--token", req.Token}, req.CLIArgs...)
	exitCode, stdout, stderr, runErr := runAgentcliCaptured(fullArgv)
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)
	_ = runErr // exit code + stderr already encode failure (Error: prefix)

	if req.WaitAfterSecs > 0 {
		time.Sleep(time.Duration(req.WaitAfterSecs) * time.Second)
	}

	if err := fillSnapshot(req, resp, m, configHome); err != nil {
		return nil, err
	}

	t.Logf("L2 service add/rm CLIArgs=%v exit=%d", req.CLIArgs, resp.ExitCode)
	return resp, nil
}

func fillSnapshot(req *Request, resp *Response, m *services.Manager, configHome string) error {
	listed := m.ListAll()
	svcs := make([]serviceStatus, 0, len(listed))
	names := make([]string, 0, len(listed))
	ids := make([]string, 0, len(listed))
	for i := range listed {
		s := toServiceStatus(&listed[i])
		svcs = append(svcs, s)
		names = append(names, s.Name)
		ids = append(ids, s.ID)
	}
	resp.ServicesAfter = svcs
	resp.ListedNames = names
	resp.ListedIDs = ids

	onDisk, err := readServicesJSON(configHome)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if onDisk == nil {
		onDisk = []map[string]any{}
	}
	resp.ServicesOnDisk = onDisk

	// Resolve target: explicit id/name, or first match after add.
	targetID := req.TargetID
	targetName := req.TargetName
	if targetID == "" && targetName != "" {
		for _, s := range svcs {
			if s.Name == targetName {
				targetID = s.ID
				break
			}
		}
		// also search disk (rm may have removed it from ListAll)
		if targetID == "" {
			for _, row := range onDisk {
				if name, _ := row["name"].(string); name == targetName {
					if id, _ := row["id"].(string); id != "" {
						targetID = id
					}
					break
				}
			}
		}
	}
	if targetID == "" && targetName == "" && len(svcs) == 1 {
		targetID = svcs[0].ID
		targetName = svcs[0].Name
	}
	resp.TargetID = targetID
	resp.TargetName = targetName
	if targetName == "" {
		resp.TargetName = req.TargetName
	}

	for _, s := range svcs {
		if s.ID == targetID || (targetName != "" && s.Name == targetName) {
			resp.TargetPID = s.PID
			resp.TargetRunning = serviceIsRunning(s)
			v := s.Enabled
			resp.TargetEnabled = &v
			if resp.TargetID == "" {
				resp.TargetID = s.ID
			}
			break
		}
	}
	if resp.TargetEnabled == nil && targetID != "" {
		for _, row := range onDisk {
			id, _ := row["id"].(string)
			if id != targetID {
				continue
			}
			if enabled, ok := row["enabled"].(bool); ok {
				v := enabled
				resp.TargetEnabled = &v
			}
			break
		}
	}
	return nil
}

func toServiceStatus(st *services.ServiceStatus) serviceStatus {
	if st == nil {
		return serviceStatus{}
	}
	return serviceStatus{
		ID: st.ID, Name: st.Name, Command: st.Command,
		ProjectDir: st.ProjectDir, WorkingDir: st.WorkingDir,
		Status: st.Status, PID: st.PID,
		DesiredRunning: st.DesiredRunning, Enabled: st.Enabled,
	}
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

	stdout = string(outBytes)
	stderr = string(errBytes)
	exitCode = 0
	if runErr != nil {
		// Match product CLI: errors surface as "Error: …" on stderr with non-zero exit.
		if !strings.Contains(stderr, "Error:") {
			stderr += fmt.Sprintf("Error: %v\n", runErr)
		}
		exitCode = 1
	}
	return exitCode, stdout, stderr, nil
}

func writeServicesJSON(configHome string, servicesList []ServiceSeed) error {
	now := "2026-07-30T00:00:00Z"
	rows := make([]servicesFileRow, 0, len(servicesList))
	for _, svc := range servicesList {
		if svc.ID == "" || svc.Name == "" || svc.Command == "" {
			return fmt.Errorf("service id, name, and command are required")
		}
		rows = append(rows, servicesFileRow{
			ID:         svc.ID,
			Name:       svc.Name,
			Command:    svc.Command,
			ProjectDir: svc.ProjectDir,
			WorkingDir: svc.WorkingDir,
			Enabled:    svc.Enabled,
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

func boolPtr(v bool) *bool { return &v }

func sleepService(id, name, projectDir string) ServiceSeed {
	return ServiceSeed{
		ID:         id,
		Name:       name,
		Command:    "sleep 300",
		ProjectDir: projectDir,
	}
}

func diskHasName(rows []map[string]any, name string) bool {
	for _, row := range rows {
		if n, _ := row["name"].(string); n == name {
			return true
		}
	}
	return false
}

func diskHasID(rows []map[string]any, id string) bool {
	for _, row := range rows {
		if rowID, _ := row["id"].(string); rowID == id {
			return true
		}
	}
	return false
}

func diskEnabled(rows []map[string]any, name string) (enabled *bool, present bool) {
	for _, row := range rows {
		if n, _ := row["name"].(string); n != name {
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

func listContainsName(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func listContainsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
```
