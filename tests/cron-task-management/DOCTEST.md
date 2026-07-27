# Cron Task Management Doctests

End-to-end contract for **cron task management** on the ai-critic user server:
scheduled shell commands (interval finish-based or 5-field UTC cron), HTTP API,
and `remote-agent cron` / `local-agent cron` CLI (same agentcli path).

No frontend UI. Global scope only. Isolated `AI_CRITIC_HOME` per leaf.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `crontasks.NewManagerAt` + `RegisterAPIWithManager`
(+ optional `agentcli.Run`). Two sparse **L3 e2e** smokes keep the product binary
path (`UseBinary` + `label: heavy, e2e`): `management/create-interval-and-list` and
`cli-schedule/cron-local-safe-convert`.

**Participants**

- **L2: crontasks.Manager** — isolated config path; tick loop via `Manager.Start`.
- **L2: httptest + agentcli.Run** — CLI schedule leaves without product binaries.
- **L3: ai-critic-server + remote-agent subprocesses** — sparse binary smokes.
- **ai-critic-server (L3)** — loads `{AI_CRITIC_HOME}/cron-tasks.json`, runs
  a background tick loop (~1s), exposes `/api/cron-tasks*`, appends run logs under
  data dir `cron-tasks/<id>.log`.
- **Cron task definitions** — global rows with `scheduleMode` (`interval` | `cron`),
  command (`bash -lc`), optional `timeout` (default `1h`, always enforced, must be >0),
  `enabled` (default true when absent).
- **Cron task manager** — schedules fires; interval next fire = **last finish + N**;
  overlap policy is **skip only** (never start a second instance while previous is live).
- **Run history** — last **7 days** of `CronTaskRun` records (API `GET …/history`);
  prune on write/tick.
- **HTTP client** — authenticated CRUD, enable/disable, manual run, history.
- **remote-agent subprocess** — `cron` subcommands; `--every` | `--cron` (local→UTC)
  | `--cron-utc` (pass-through); timestamps displayed in local TZ.
- **Test harness** — session-cached binaries, per-leaf config home + work dir for
  marker files, free port, `lib.TestPassword` credentials.

**Behaviors**

- Create/list/update/delete definitions; list is always global (no projectDir/`?all=1`).
- Interval tasks fire after prior run **finishes** plus interval; first fire soon after
  enable/create when due.
- If still running at fire time → skip (no second process).
- Timeout kills process group; status/history record timeout error.
- Disabled tasks are never scheduled; enable restores scheduling.
- Manual `run` fires immediately unless already running (then skip).
- Cron expressions stored and evaluated in **UTC** on the server.
- CLI `--cron` converts local wall time to UTC when **safe**; unsafe patterns or
  ambiguous DST rewrites **error** and must mention `--cron-utc`.
- CLI `--cron-utc` stores expression as-is.
- Validation rejects missing schedule mode, both modes at once, and non-positive timeout.

## Version

0.0.3

## Decision Tree

```
[cron task management — server API + remote-agent cron CLI]
 |
 +-- management/                         (GROUP)  definition CRUD
 |    +-- list-empty/                    (LEAF)   empty home → list []
 |    +-- create-interval-and-list/      (LEAF)   POST create interval → list shows it
 |    +-- remove-deletes/                (LEAF)   delete → list no longer contains id
 |
 +-- runtime/                            (GROUP)  scheduler + execution rules
 |    +-- interval-finish-based/         (LEAF)   short interval runs; next after finish+N
 |    +-- overlap-skip/                  (LEAF)   long sleep + short interval → one instance
 |    +-- timeout-enforced/              (LEAF)   sleep past timeout → killed + error
 |    +-- default-timeout-1h/            (LEAF)   omit timeout → definition uses 1h
 |    +-- enable-disable-gating/         (LEAF)   disabled never fires; enable restores
 |    +-- manual-run/                    (LEAF)   POST/CLI run fires before next schedule
 |    +-- cron-utc-storage/              (LEAF)   cron expr stored/evaluated as UTC
 |
 +-- history/                            (GROUP)  7-day run history
 |    +-- returns-recent-runs/           (LEAF)   after runs, history lists them (UTC)
 |    +-- prunes-older-than-7d/          (LEAF)   seeded run older than 7d is pruned
 |
 +-- cli-schedule/                       (GROUP)  CLI schedule flags / convert
 |    +-- cron-local-safe-convert/       (LEAF)   --cron fixed-offset → UTC + both printed
 |    +-- cron-local-unsafe-error/       (LEAF)   unsafe --cron → non-zero, mentions --cron-utc
 |
 +-- validation/                         (GROUP)  create/update reject invalid inputs
      +-- both-schedule-modes/           (LEAF)   interval + cron together → error
      +-- neither-schedule-mode/         (LEAF)   neither schedule → error
      +-- timeout-non-positive/          (LEAF)   timeout ≤0 → error
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `management/list-empty` | Fresh home: `GET /api/cron-tasks` returns `[]` |
| 2 | `management/create-interval-and-list` | Create interval task; list includes name/command/mode |
| 3 | `management/remove-deletes` | Delete by id; subsequent list omits task |
| 4 | `runtime/interval-finish-based` | Short interval + marker command runs ≥2 times; `nextRunAt` ≥ finish+interval |
| 5 | `runtime/overlap-skip` | Long-running command still live when due → no second start |
| 6 | `runtime/timeout-enforced` | Timeout shorter than sleep → process killed; history/status error |
| 7 | `runtime/default-timeout-1h` | Create without timeout field → status/definition timeout is `1h` |
| 8 | `runtime/enable-disable-gating` | Disabled: no scheduled fires; after enable, fires occur |
| 9 | `runtime/manual-run` | Long interval; `POST …/run` produces a run immediately |
| 10 | `runtime/cron-utc-storage` | Create with UTC cron expr; list stores same `cronExpr` |
| 11 | `history/returns-recent-runs` | After forced runs, `GET …/history` returns recent entries |
| 12 | `history/prunes-older-than-7d` | Seeded run >7d old is absent after prune/list/history |
| 13 | `cli-schedule/cron-local-safe-convert` | Fixed-offset TZ + `--cron` converts; stdout shows local + UTC |
| 14 | `cli-schedule/cron-local-unsafe-error` | Complex/unsafe `--cron` exits non-zero; message mentions `--cron-utc` |
| 15 | `validation/both-schedule-modes` | Body with both interval and cron rejected |
| 16 | `validation/neither-schedule-mode` | Body with neither schedule mode rejected |
| 17 | `validation/timeout-non-positive` | `timeout: "0"` (or ≤0) rejected |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| CRUD list empty / create / delete | management/* |
| Interval finish-based scheduling | runtime/interval-finish-based |
| Overlap = skip only | runtime/overlap-skip |
| Timeout always enforced | runtime/timeout-enforced |
| Default timeout 1h | runtime/default-timeout-1h |
| Enable/disable gating | runtime/enable-disable-gating |
| Manual run | runtime/manual-run |
| Cron UTC on server | runtime/cron-utc-storage |
| History 7d | history/* |
| CLI `--cron` safe convert | cli-schedule/cron-local-safe-convert |
| CLI `--cron` unsafe → error | cli-schedule/cron-local-unsafe-error |
| Validation XOR schedule + timeout >0 | validation/* |
| Global scope only | all leaves (no projectDir) |
| No UI | N/A (server+CLI only) |

## How to Run

```sh
go run ./script/build   # optional; harness builds session-cached binaries
doctest vet ./tests/cron-task-management
doctest test ./tests/cron-task-management/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/crontasks"
	"github.com/xhd2015/doctest/session"
)

var agentcliInProcessMu sync.Mutex


// TaskSeed is one row written to cron-tasks.json before the server starts.
type TaskSeed struct {
	ID           string
	Name         string
	Command      string
	WorkingDir   string
	ScheduleMode string // "interval" | "cron"
	Interval     string
	CronExpr     string
	Enabled      *bool
	Timeout      string
	// RecentRuns optional history rows for prune tests (JSON-shaped maps).
	RecentRuns []map[string]any
}

// Request configures one leaf: seed, primary action (API or CLI), and observation.
type Request struct {
	SeedTasks []TaskSeed

	// Action after server is ready:
	// list | create | update | delete | enable | disable | run | history | none
	// "none" only observes (wait/poll after seeds).
	Action string

	// Definition fields for create/update (API body or CLI flags when UseCLI).
	TaskName     string
	Command      string
	WorkingDir   string
	ScheduleMode string // "interval" | "cron"
	Interval     string
	CronExpr     string
	Timeout      string // empty = omit field (server default 1h)
	Enabled      *bool

	// When set, POST/PUT this raw JSON object instead of building from fields
	// (validation leaves).
	RawBody map[string]any

	// Target name or id for update/delete/enable/disable/run/history.
	Target string

	UseCLI  bool
	CLIArgs []string
	// Extra process env for CLI (e.g. "TZ=Etc/GMT-8").
	CLIEnv []string

	// UseBinary / E2E force L3 product binaries. Default false → L2 Manager/agentcli.
	UseBinary bool
	E2E       bool

	// Pre-phase wait after seed boot (before main action), for disable gating etc.
	PreWaitSecs int
	// Capture list+history after PreWaitSecs into Response.Pre*.
	CapturePreSnapshot bool

	// Post-action observation.
	WaitSecs        int
	PollRunsMin     int // poll history until ≥N runs (0 = skip)
	PollTimeoutSecs int // default 20

	// When true, command in create is rewritten to append a line to WorkDir/marker.txt
	// (absolute path). Use with short intervals to prove fires.
	UseMarker bool

	// After create via API, also call list (always done). Optional second wait.
	Token      string
	ServerPort int
}

type cronTaskRun struct {
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt,omitempty"`
	ExitCode   *int   `json:"exitCode,omitempty"`
	Error      string `json:"error,omitempty"`
}

type cronTaskStatus struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Command        string        `json:"command"`
	WorkingDir     string        `json:"workingDir,omitempty"`
	ScheduleMode   string        `json:"scheduleMode"`
	Interval       string        `json:"interval,omitempty"`
	CronExpr       string        `json:"cronExpr,omitempty"`
	Enabled        bool          `json:"enabled"`
	Timeout        string        `json:"timeout,omitempty"`
	Status         string        `json:"status"`
	PID            int           `json:"pid,omitempty"`
	LastStartedAt  string        `json:"lastStartedAt,omitempty"`
	LastFinishedAt string        `json:"lastFinishedAt,omitempty"`
	LastExitCode   *int          `json:"lastExitCode,omitempty"`
	LastError      string        `json:"lastError,omitempty"`
	NextRunAt      string        `json:"nextRunAt,omitempty"`
	LogPath        string        `json:"logPath"`
	RecentRuns     []cronTaskRun `json:"recentRuns,omitempty"`
	CreatedAt      string        `json:"createdAt,omitempty"`
	UpdatedAt      string        `json:"updatedAt,omitempty"`
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ConfigHome string
	WorkDir    string
	AgentHome  string
	MarkerPath string

	HTTPStatus  int
	ActionError string
	Body        string

	Tasks   []cronTaskStatus
	Target  *cronTaskStatus
	History []cronTaskRun

	// Pre-action snapshot (when CapturePreSnapshot).
	PreTasks   []cronTaskStatus
	PreHistory []cronTaskRun
	PreRuns    int

	// Evidence
	MarkerContent string
	MarkerLines   int
	TargetPID     int
	ProcessAlive  bool
	RunCount      int
}

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseBinary || req.E2E)
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if req.PollTimeoutSecs <= 0 {
		req.PollTimeoutSecs = 20
	}
	if req.Action == "" {
		req.Action = "list"
	}

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	workDir, err := os.MkdirTemp("", "cron-task-work-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(workDir) })
	resp.WorkDir = workDir
	resp.MarkerPath = filepath.Join(workDir, "marker.txt")

	agentHome, err := os.MkdirTemp("", "remote-agent-cron-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome
	aiCriticAgent := filepath.Join(agentHome, ".ai-critic")
	if err := os.MkdirAll(aiCriticAgent, 0755); err != nil {
		return nil, err
	}

	if req.UseMarker {
		if req.Command == "" {
			req.Command = fmt.Sprintf("echo ran >> %q", resp.MarkerPath)
		} else if !strings.Contains(req.Command, resp.MarkerPath) {
			req.Command = fmt.Sprintf("(%s); echo ran >> %q", req.Command, resp.MarkerPath)
		}
	}

	if len(req.SeedTasks) > 0 {
		if err := writeCronTasksJSON(configHome, req.SeedTasks); err != nil {
			return nil, err
		}
	}

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, configHome, agentHome, aiCriticAgent)
	}
	return runInProcessL2(t, req, resp, configHome, agentHome, aiCriticAgent)
}

func runInProcessL2(t *testing.T, req *Request, resp *Response, configHome, agentHome, aiCriticAgent string) (*Response, error) {
	cfgPath := filepath.Join(configHome, "cron-tasks.json")
	m := crontasks.NewManagerAt(cfgPath)
	m.Start()
	t.Cleanup(m.Shutdown)

	mux := http.NewServeMux()
	crontasks.RegisterAPIWithManager(mux, m)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			mux.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+req.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	baseURL := srv.URL
	resp.ServerPort = 0

	if req.PreWaitSecs > 0 {
		time.Sleep(time.Duration(req.PreWaitSecs) * time.Second)
	}
	if req.CapturePreSnapshot {
		tasks, err := getCronTasks(baseURL, req.Token)
		if err != nil {
			return nil, fmt.Errorf("pre list: %w", err)
		}
		resp.PreTasks = tasks
		if id := resolveTargetID(req, tasks); id != "" {
			hist, err := getCronHistory(baseURL, req.Token, id)
			if err == nil {
				resp.PreHistory = hist
				resp.PreRuns = len(hist)
			}
		}
	}

	if req.UseCLI {
		if err := writeRemoteAgentConfig(filepath.Join(aiCriticAgent, "remote-agent-config.json"), baseURL, req.Token); err != nil {
			return nil, err
		}
		argv := req.CLIArgs
		if len(argv) == 0 {
			argv = buildCLIArgs(req)
		}
		fullArgv := append([]string{"--server", baseURL, "--token", req.Token}, argv...)
		// CLIEnv: agentcli uses process env for TZ; mutate under mutex only (not Setenv API).
		exitCode, stdout, stderr, _ := runAgentcliCapturedWithEnv(fullArgv, req.CLIEnv)
		resp.ExitCode = exitCode
		resp.Stdout = stdout
		resp.Stderr = stderr
		resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)
	} else {
		switch req.Action {
		case "none", "list":
			// observation only
		case "create":
			body := req.RawBody
			if body == nil {
				body = buildCreateBody(req)
			}
			status, raw, created, err := postCronCreate(baseURL, req.Token, body)
			resp.HTTPStatus = status
			resp.Body = raw
			if err != nil {
				resp.ActionError = err.Error()
			} else if created != nil {
				resp.Target = created
				if req.Target == "" {
					req.Target = created.ID
				}
			}
		case "update":
			body := req.RawBody
			if body == nil {
				body = buildCreateBody(req)
			}
			if req.Target != "" {
				body["id"] = req.Target
			}
			status, raw, updated, err := putCronUpdate(baseURL, req.Token, body)
			resp.HTTPStatus = status
			resp.Body = raw
			if err != nil {
				resp.ActionError = err.Error()
			} else if updated != nil {
				resp.Target = updated
			}
		case "delete", "enable", "disable", "run":
			id := req.Target
			if id == "" && resp.Target != nil {
				id = resp.Target.ID
			}
			if id == "" {
				tasks, _ := getCronTasks(baseURL, req.Token)
				id = resolveTargetID(req, tasks)
			}
			switch req.Action {
			case "delete":
				status, raw, err := deleteCronTask(baseURL, req.Token, id)
				resp.HTTPStatus = status
				resp.Body = raw
				if err != nil {
					resp.ActionError = err.Error()
				}
			case "enable":
				status, raw, err := postCronAction(baseURL, req.Token, "/api/cron-tasks/enable", id)
				resp.HTTPStatus = status
				resp.Body = raw
				if err != nil {
					resp.ActionError = err.Error()
				}
			case "disable":
				status, raw, err := postCronAction(baseURL, req.Token, "/api/cron-tasks/disable", id)
				resp.HTTPStatus = status
				resp.Body = raw
				if err != nil {
					resp.ActionError = err.Error()
				}
			case "run":
				status, raw, err := postCronAction(baseURL, req.Token, "/api/cron-tasks/run", id)
				resp.HTTPStatus = status
				resp.Body = raw
				if err != nil {
					resp.ActionError = err.Error()
				}
			}
		case "history":
			// final snapshot
		default:
			return nil, fmt.Errorf("unknown action %q", req.Action)
		}
	}

	if req.WaitSecs > 0 {
		time.Sleep(time.Duration(req.WaitSecs) * time.Second)
	}

	tasks, err := getCronTasks(baseURL, req.Token)
	if err != nil {
		actionFailed := resp.ActionError != "" ||
			resp.ExitCode != 0 ||
			(resp.HTTPStatus != 0 && (resp.HTTPStatus < 200 || resp.HTTPStatus >= 300))
		if !actionFailed {
			return nil, err
		}
		tasks = nil
	} else {
		resp.Tasks = tasks
	}

	id := resolveTargetID(req, tasks)
	if id == "" && resp.Target != nil {
		id = resp.Target.ID
	}
	if id != "" {
		for i := range tasks {
			if tasks[i].ID == id || tasks[i].Name == id {
				cp := tasks[i]
				resp.Target = &cp
				resp.TargetPID = cp.PID
				resp.ProcessAlive = processAlive(cp.PID)
				break
			}
		}
		if req.PollRunsMin > 0 {
			deadline := time.Now().Add(time.Duration(req.PollTimeoutSecs) * time.Second)
			for time.Now().Before(deadline) {
				hist, hErr := getCronHistory(baseURL, req.Token, id)
				if hErr == nil && len(hist) >= req.PollRunsMin {
					resp.History = hist
					resp.RunCount = len(hist)
					break
				}
				if hErr == nil {
					resp.History = hist
					resp.RunCount = len(hist)
				}
				time.Sleep(200 * time.Millisecond)
			}
		} else if req.Action == "history" || req.PollRunsMin == 0 {
			hist, hErr := getCronHistory(baseURL, req.Token, id)
			if hErr == nil {
				resp.History = hist
				resp.RunCount = len(hist)
			} else if req.Action == "history" && resp.ActionError == "" {
				resp.ActionError = hErr.Error()
			}
		}
		if tasks2, err2 := getCronTasks(baseURL, req.Token); err2 == nil {
			resp.Tasks = tasks2
			for i := range tasks2 {
				if tasks2[i].ID == id || tasks2[i].Name == id {
					cp := tasks2[i]
					resp.Target = &cp
					resp.TargetPID = cp.PID
					resp.ProcessAlive = processAlive(cp.PID)
					break
				}
			}
		}
	}

	if data, err := os.ReadFile(resp.MarkerPath); err == nil {
		resp.MarkerContent = string(data)
		resp.MarkerLines = countNonEmptyLines(resp.MarkerContent)
	}
	t.Logf("L2 cron action=%s useCLI=%v", req.Action, req.UseCLI)
	return resp, nil
}

func runAgentcliCapturedWithEnv(argv []string, extraEnv []string) (exitCode int, stdout, stderr string, err error) {
	// L2 path does not mutate process env (no Setenv). TZ-sensitive convert is L3-only.
	_ = extraEnv
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
		exitCode = 1
		// agentcli returns errors without always printing them.
		stderr = strings.TrimSpace(stderr + "\n" + runErr.Error())
	}
	return exitCode, stdout, stderr, runErr
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, configHome, agentHome, aiCriticAgent string) (*Response, error) {
	moduleRoot, err := findModuleRoot(d)
	if err != nil {
		return nil, err
	}

	cacheDir := sessionCacheDir(d)
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

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

	if req.PreWaitSecs > 0 {
		time.Sleep(time.Duration(req.PreWaitSecs) * time.Second)
	}
	if req.CapturePreSnapshot {
		tasks, err := getCronTasks(baseURL, req.Token)
		if err != nil {
			return nil, fmt.Errorf("pre list: %w", err)
		}
		resp.PreTasks = tasks
		if id := resolveTargetID(req, tasks); id != "" {
			hist, err := getCronHistory(baseURL, req.Token, id)
			if err == nil {
				resp.PreHistory = hist
				resp.PreRuns = len(hist)
			}
		}
	}

	switch {
	case req.UseCLI:
		serverURL := fmt.Sprintf("http://localhost:%d", serverPort)
		if err := writeRemoteAgentConfig(filepath.Join(aiCriticAgent, "remote-agent-config.json"), serverURL, req.Token); err != nil {
			return nil, err
		}
		argv := req.CLIArgs
		if len(argv) == 0 {
			argv = buildCLIArgs(req)
		}
		fullArgv := append([]string{"--server", serverURL, "--token", req.Token}, argv...)
		agentCmd := exec.Command(agentBin, fullArgv...)
		agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
		agentEnv = append(agentEnv, "HOME="+agentHome)
		for _, e := range req.CLIEnv {
			if i := strings.IndexByte(e, '='); i > 0 {
				agentEnv = stripEnvPrefix(agentEnv, e[:i+1])
			}
			agentEnv = append(agentEnv, e)
		}
		agentCmd.Env = agentEnv
		var stdout, stderr bytes.Buffer
		agentCmd.Stdout = &stdout
		agentCmd.Stderr = &stderr
		runErr := agentCmd.Run()
		if runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				resp.ExitCode = exitErr.ExitCode()
			} else {
				resp.ActionError = runErr.Error()
			}
		}
		resp.Stdout = stdout.String()
		resp.Stderr = stderr.String()
		resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	case req.Action == "none", req.Action == "list":

	case req.Action == "create":
		body := req.RawBody
		if body == nil {
			body = buildCreateBody(req)
		}
		status, raw, created, err := postCronCreate(baseURL, req.Token, body)
		resp.HTTPStatus = status
		resp.Body = raw
		if err != nil {
			resp.ActionError = err.Error()
		} else if created != nil {
			resp.Target = created
			if req.Target == "" {
				req.Target = created.ID
			}
		}

	case req.Action == "update":
		body := req.RawBody
		if body == nil {
			body = buildCreateBody(req)
		}
		if req.Target != "" {
			body["id"] = req.Target
		}
		status, raw, updated, err := putCronUpdate(baseURL, req.Token, body)
		resp.HTTPStatus = status
		resp.Body = raw
		if err != nil {
			resp.ActionError = err.Error()
		} else if updated != nil {
			resp.Target = updated
		}

	case req.Action == "delete", req.Action == "enable", req.Action == "disable", req.Action == "run":
		id := req.Target
		if id == "" && resp.Target != nil {
			id = resp.Target.ID
		}
		if id == "" {
			tasks, _ := getCronTasks(baseURL, req.Token)
			id = resolveTargetID(req, tasks)
		}
		switch req.Action {
		case "delete":
			status, raw, err := deleteCronTask(baseURL, req.Token, id)
			resp.HTTPStatus = status
			resp.Body = raw
			if err != nil {
				resp.ActionError = err.Error()
			}
		case "enable":
			status, raw, err := postCronAction(baseURL, req.Token, "/api/cron-tasks/enable", id)
			resp.HTTPStatus = status
			resp.Body = raw
			if err != nil {
				resp.ActionError = err.Error()
			}
		case "disable":
			status, raw, err := postCronAction(baseURL, req.Token, "/api/cron-tasks/disable", id)
			resp.HTTPStatus = status
			resp.Body = raw
			if err != nil {
				resp.ActionError = err.Error()
			}
		case "run":
			status, raw, err := postCronAction(baseURL, req.Token, "/api/cron-tasks/run", id)
			resp.HTTPStatus = status
			resp.Body = raw
			if err != nil {
				resp.ActionError = err.Error()
			}
		}

	case req.Action == "history":

	default:
		return nil, fmt.Errorf("unknown action %q", req.Action)
	}

	if req.WaitSecs > 0 {
		time.Sleep(time.Duration(req.WaitSecs) * time.Second)
	}

	tasks, err := getCronTasks(baseURL, req.Token)
	if err != nil {
		actionFailed := resp.ActionError != "" ||
			resp.ExitCode != 0 ||
			(resp.HTTPStatus != 0 && (resp.HTTPStatus < 200 || resp.HTTPStatus >= 300))
		if !actionFailed {
			return nil, err
		}
		tasks = nil
	} else {
		resp.Tasks = tasks
	}

	id := resolveTargetID(req, tasks)
	if id == "" && resp.Target != nil {
		id = resp.Target.ID
	}
	if id != "" {
		for i := range tasks {
			if tasks[i].ID == id || tasks[i].Name == id {
				cp := tasks[i]
				resp.Target = &cp
				resp.TargetPID = cp.PID
				resp.ProcessAlive = processAlive(cp.PID)
				break
			}
		}
		if req.PollRunsMin > 0 {
			deadline := time.Now().Add(time.Duration(req.PollTimeoutSecs) * time.Second)
			for time.Now().Before(deadline) {
				hist, hErr := getCronHistory(baseURL, req.Token, id)
				if hErr == nil && len(hist) >= req.PollRunsMin {
					resp.History = hist
					resp.RunCount = len(hist)
					break
				}
				if hErr == nil {
					resp.History = hist
					resp.RunCount = len(hist)
				}
				time.Sleep(200 * time.Millisecond)
			}
		} else if req.Action == "history" || req.PollRunsMin == 0 {
			hist, hErr := getCronHistory(baseURL, req.Token, id)
			if hErr == nil {
				resp.History = hist
				resp.RunCount = len(hist)
			} else if req.Action == "history" && resp.ActionError == "" {
				resp.ActionError = hErr.Error()
			}
		}
		if tasks2, err2 := getCronTasks(baseURL, req.Token); err2 == nil {
			resp.Tasks = tasks2
			for i := range tasks2 {
				if tasks2[i].ID == id || tasks2[i].Name == id {
					cp := tasks2[i]
					resp.Target = &cp
					resp.TargetPID = cp.PID
					resp.ProcessAlive = processAlive(cp.PID)
					break
				}
			}
		}
	}

	if data, err := os.ReadFile(resp.MarkerPath); err == nil {
		resp.MarkerContent = string(data)
		resp.MarkerLines = countNonEmptyLines(resp.MarkerContent)
	}
	return resp, nil
}

func buildCreateBody(req *Request) map[string]any {
	body := map[string]any{}
	if req.TaskName != "" {
		body["name"] = req.TaskName
	}
	if req.Command != "" {
		body["command"] = req.Command
	}
	if req.WorkingDir != "" {
		body["workingDir"] = req.WorkingDir
	}
	if req.ScheduleMode != "" {
		body["scheduleMode"] = req.ScheduleMode
	}
	if req.Interval != "" {
		body["interval"] = req.Interval
	}
	if req.CronExpr != "" {
		body["cronExpr"] = req.CronExpr
	}
	if req.Timeout != "" {
		body["timeout"] = req.Timeout
	}
	if req.Enabled != nil {
		body["enabled"] = *req.Enabled
	}
	return body
}

func buildCLIArgs(req *Request) []string {
	switch req.Action {
	case "list":
		return []string{"cron", "list"}
	case "create":
		args := []string{"cron", "add", "--name", req.TaskName, "--command", req.Command}
		if req.Interval != "" {
			args = append(args, "--every", req.Interval)
		}
		if req.CronExpr != "" {
			args = append(args, "--cron-utc", req.CronExpr)
		}
		if req.Timeout != "" {
			args = append(args, "--timeout", req.Timeout)
		}
		if req.WorkingDir != "" {
			args = append(args, "--working-dir", req.WorkingDir)
		}
		if req.Enabled != nil && !*req.Enabled {
			args = append(args, "--disabled")
		}
		return args
	case "delete":
		return []string{"cron", "remove", req.Target}
	case "enable":
		return []string{"cron", "enable", req.Target}
	case "disable":
		return []string{"cron", "disable", req.Target}
	case "run":
		return []string{"cron", "run", req.Target}
	case "history":
		return []string{"cron", "history", req.Target}
	default:
		return []string{"cron", "list"}
	}
}

func resolveTargetID(req *Request, tasks []cronTaskStatus) string {
	if req.Target != "" {
		for _, t := range tasks {
			if t.ID == req.Target || t.Name == req.Target {
				return t.ID
			}
		}
		return req.Target
	}
	if req.TaskName != "" {
		for _, t := range tasks {
			if t.Name == req.TaskName {
				return t.ID
			}
		}
	}
	if len(tasks) == 1 {
		return tasks[0].ID
	}
	return ""
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

```
