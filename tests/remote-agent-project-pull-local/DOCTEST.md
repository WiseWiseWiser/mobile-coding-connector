# Remote-Agent Project bind-local & pull-local Doctests

Doctests for `remote-agent project bind-local` and `remote-agent project pull-local`:
origin validation, config bindings, dirty-state transfer into local git worktrees,
flags, and submodule guards.

Most leaves are **L2 in-process** (`projects.RegisterAPIForFile` +
`projectpull.RegisterAPI` + `agentcli.Run` with testhooks home override). Two
sparse **L3 e2e** smokes keep the product binary path.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: projects + projectpull HTTP handlers on a local
mux + `agentcli.Run` with mutex-scoped home override (no product binaries, no
process Setenv). Two sparse **L3 e2e** smokes keep the binary path
(`UseCLI` + `label: heavy, e2e`): `pull-local/bound-dirty-success` and
`pull-local/size-limits/oversized-file-rejected`.

For `pull-local`, the server exposes `POST /api/remote-agent/project/pull-local`
(dry-run JSON plan or streamed tar.gz package) and
`POST /api/remote-agent/project/pull-local/truncate`; the CLI applies the package
locally. `bind-local` remains local-config only.

**Participants**

- **L2: agentcli.Run** — in-process `bind-local` / `pull-local` against the local mux.
- **L2: projects + projectpull HTTP** — `RegisterAPIForFile` + `projectpull.RegisterAPI`
  on ephemeral port (same wire paths as production).
- **L3: remote-agent + ai-critic-server subprocesses** — sparse `UseCLI` smokes.
- **Remote project directory** — temp clone registered as `ProjectInfo.Dir`.
- **Local git repository** — separate temp clone sharing a `file://` bare origin.
- **Isolated agent HOME** — config + `project-worktrees/` via testhooks override (L2)
  or child `HOME=` (L3).

**Behaviors**

- `bind-local` upserts `(server, remote_dir) → local_path` after same-origin check.
- `pull-local` refuses clean remotes, missing bindings on non-TTY stdin, origin
  mismatch, and dirty submodules (including before `--dry-run` plan output).
- Successful pull calls the server package endpoint, creates a **named-branch**
  worktree at the computed slug path (e.g. branch `main-1` for directory `main-1`,
  not detached HEAD), applies `patch.diff` and untracked members from the tarball,
  and by default truncates the remote repo via the truncate API.
- Per-file and total byte limits reject oversized pulls unless
  `--include-file` or `--max-size` overrides apply.
- `--no-truncate-remote` leaves remote porcelain intact; `--dry-run` prints a plan
  without worktree or remote mutations.
- Repeated pulls on the same branch allocate incrementing worktree suffixes (`main-1`,
  `main-2`, …).

## Version

0.0.3

## Decision Tree

```
[remote-agent project bind-local | pull-local]
 |
 +-- bind-local/                         (GROUP)  origin + local repo validation
 |    |
 |    +-- same-origin/                   (LEAF)   matching file:// origin → binding saved
 |    +-- origin-mismatch/               (LEAF)   different origins → exit 1
 |    +-- not-git-repo/                  (LEAF)   local path not git → exit 1
 |
 +-- pull-local/                         (GROUP)  dirty transfer + flags + submodules
      |
      +-- bound-dirty-success/           (LEAF)   binding + dirty → worktree + remote clean
      +-- clean-remote/                  (LEAF)   clean remote → exit 1
      +-- no-binding-non-tty/            (LEAF)   piped stdin, no binding → exit 1
      +-- no-truncate-remote/            (LEAF)   flag keeps remote dirty
      +-- dry-run/                       (LEAF)   plan only, no worktree, remote unchanged
      +-- worktree-collision/            (LEAF)   two pulls → main-2 when main-1 exists; both on named branches
      +-- worktree-on-branch/            (LEAF)   symbolic-ref HEAD; branch matches worktree suffix
      +-- submodule-clean/               (LEAF)   clean submodule, dirty top-level → exit 0
      +-- submodule-dirty/               (LEAF)   dirty submodule path in error
      +-- dry-run-submodule-dirty/       (LEAF)   dry-run blocked before plan
      |
      +-- size-limits/                   (GROUP)  server byte caps + CLI overrides
           |
           +-- oversized-file-rejected/  (LEAF)   2MB untracked, no --include-file → exit 1
           +-- include-file-allows-large/ (LEAF)   same + --include-file big.bin → exit 0
           +-- include-file-not-dirty/   (LEAF)   --include-file not in dirty set → exit 1
           +-- total-over-max/           (LEAF)   >64MB dirty set → exit 1, --max-size hint
           +-- max-size-override/        (LEAF)   same + --max-size 100M → exit 0
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `bind-local/same-origin` | Same bare origin; binding persisted in config |
| 2 | `bind-local/origin-mismatch` | Local vs remote origin differ; mismatch error |
| 3 | `bind-local/not-git-repo` | Local path is not a git repo |
| 4 | `pull-local/bound-dirty-success` | Seeded binding; modified + untracked pulled; remote clean |
| 5 | `pull-local/clean-remote` | Nothing to pull when worktree clean |
| 6 | `pull-local/no-binding-non-tty` | Non-TTY without binding or `--local-path` |
| 7 | `pull-local/no-truncate-remote` | Worktree ok; remote stays dirty |
| 8 | `pull-local/dry-run` | Exit 0 plan; no worktree dir; remote still dirty |
| 9 | `pull-local/worktree-collision` | Second pull uses `main-2`; each worktree on matching named branch |
| 18 | `pull-local/worktree-on-branch` | `git symbolic-ref HEAD` succeeds; `branch --show-current` = `main-1` |
| 10 | `pull-local/submodule-clean` | Dirty top-level with clean submodule succeeds |
| 11 | `pull-local/submodule-dirty` | Dirty file inside submodule fails |
| 12 | `pull-local/dry-run-submodule-dirty` | Submodule guard before dry-run plan |
| 13 | `pull-local/size-limits/oversized-file-rejected` | 2MB file over 1MB cap without include |
| 14 | `pull-local/size-limits/include-file-allows-large` | `--include-file` exempts large untracked |
| 15 | `pull-local/size-limits/include-file-not-dirty` | Include path not part of pull |
| 16 | `pull-local/size-limits/total-over-max` | Package over default 64MB total |
| 17 | `pull-local/size-limits/max-size-override` | `--max-size 100M` allows large package |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Subcommand `bind-local` | bind-local/* |
| Subcommand `pull-local` | pull-local/* |
| Same vs mismatched origin | same-origin, origin-mismatch |
| Local path not git | not-git-repo |
| Remote dirty vs clean | bound-dirty-success, clean-remote, dry-run |
| Binding present vs absent | bound-dirty-success, no-binding-non-tty |
| Non-TTY stdin | no-binding-non-tty |
| `--no-truncate-remote` | no-truncate-remote |
| `--dry-run` | dry-run, dry-run-submodule-dirty |
| Worktree suffix allocation | worktree-collision |
| Named branch (not detached) | bound-dirty-success, worktree-collision, worktree-on-branch |
| Submodule clean vs dirty | submodule-clean, submodule-dirty, dry-run-submodule-dirty |
| Per-file 1MB cap | oversized-file-rejected, include-file-allows-large |
| `--include-file` valid vs invalid | include-file-allows-large, include-file-not-dirty |
| Total 64MB cap vs `--max-size` | total-over-max, max-size-override |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-project-pull-local
doctest test ./tests/remote-agent-project-pull-local/...          # unlabeled L2 mass
doctest test --label e2e ./tests/remote-agent-project-pull-local/...  # ~2 L3 smokes
go test ./server/projectpull/... ./cmd/agentcli/... -count=1
```

Submodule leaves need `GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=protocol.file.allow GIT_CONFIG_VALUE_0=always` (set in root `Run` agent env).

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
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/script/lib"
	serverexec "github.com/xhd2015/ai-critic/server/exec"
	"github.com/xhd2015/ai-critic/server/projectpull"
	"github.com/xhd2015/ai-critic/server/projects"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run (profile + I/O + home override).
var agentcliInProcessMu sync.Mutex

// ProjectEntry is one row written to projects.json before the CLI runs.
type ProjectEntry struct {
	ID              string
	Name            string
	Dir             string
	GitUserConfigID string
	GitUserName     string
	GitUserEmail    string
}

// ProjectBinding mirrors remote-agent-config.json project_bindings rows.
type ProjectBinding struct {
	Server    string
	RemoteDir string
	LocalPath string
}

type Request struct {
	Args   []string
	Server string
	Token  string

	Project  ProjectEntry
	Projects []ProjectEntry

	LocalPath    string
	SeedBindings []ProjectBinding
	PipeStdin    bool

	WorktreeCollision bool

	RemoteDirAfterSetup string

	// UseCLI forces the L3 product-binary path. Default false → L2 in-process.
	UseCLI bool
	E2E    bool
}

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseCLI || req.E2E)
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ConfigHome string
	AgentHome  string
	ProjectDir string
	LocalPath  string

	RemoteConfigPath string
	InvocationCount  int
}

type projectsFileRow struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	RepoURL         string `json:"repo_url"`
	Dir             string `json:"dir"`
	GitUserConfigID string `json:"git_user_config_id,omitempty"`
	GitUserName     string `json:"git_user_name,omitempty"`
	GitUserEmail    string `json:"git_user_email,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type remoteAgentConfigFile struct {
	Default         string            `json:"default,omitempty"`
	Domains         []domainConfigRow `json:"domains"`
	ProjectBindings []bindingRow      `json:"project_bindings,omitempty"`
}

type domainConfigRow struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

type bindingRow struct {
	Server    string `json:"server"`
	RemoteDir string `json:"remote_dir"`
	LocalPath string `json:"local_path"`
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	agentHome, err := os.MkdirTemp("", "remote-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	aiCriticAgent := filepath.Join(agentHome, ".ai-critic")
	if err := os.MkdirAll(aiCriticAgent, 0755); err != nil {
		return nil, err
	}
	resp.RemoteConfigPath = filepath.Join(aiCriticAgent, "remote-agent-config.json")

	projectsList := req.Projects
	if len(projectsList) == 0 && req.Project.Dir != "" {
		projectsList = []ProjectEntry{req.Project}
	}
	if len(projectsList) > 0 {
		for i := range projectsList {
			absDir, err := filepath.Abs(projectsList[i].Dir)
			if err != nil {
				return nil, fmt.Errorf("abs project dir: %w", err)
			}
			projectsList[i].Dir = absDir
		}
		req.Projects = projectsList
		resp.ProjectDir = projectsList[0].Dir
		req.RemoteDirAfterSetup = projectsList[0].Dir
		if err := writeProjectsJSON(configHome, projectsList); err != nil {
			return nil, err
		}
	}

	if req.LocalPath != "" {
		absLocal, err := filepath.Abs(req.LocalPath)
		if err != nil {
			return nil, err
		}
		req.LocalPath = absLocal
		resp.LocalPath = absLocal
	}

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, moduleRoot, cacheDir, configHome, agentHome)
	}
	return runInProcessL2(t, d, req, resp, configHome, agentHome)
}

func withBearerAuth(validToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != validToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func runInProcessL2(t *testing.T, d *session.Doctest, req *Request, resp *Response, configHome, agentHome string) (*Response, error) {
	t.Helper()

	mux := http.NewServeMux()
	projects.RegisterAPIForFile(mux, filepath.Join(configHome, "projects.json"))
	projectpull.RegisterAPI(mux)
	// bind-local / pull-local resolve remote origin via POST /api/exec (git -C …).
	serverexec.RegisterAPI(mux)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := withBearerAuth(lib.TestPassword, mux)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen in-process pull-local server: %w", err)
	}
	serverPort := ln.Addr().(*net.TCPAddr).Port
	resp.ServerPort = serverPort
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	}
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	if err := writeRemoteAgentConfig(resp.RemoteConfigPath, normalizedServer, req.Token, req.SeedBindings); err != nil {
		return nil, err
	}

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 10*time.Second); err != nil {
		return nil, err
	}
	t.Logf("L2 in-process pull-local server on %s", normalizedServer)

	return finishPullRuns(t, req, resp, serverURL, agentHome, true, "")
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, moduleRoot, cacheDir, configHome, agentHome string) (*Response, error) {
	t.Helper()

	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort := pickFreePort(portBase)
	resp.ServerPort = serverPort

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	if err := writeRemoteAgentConfig(resp.RemoteConfigPath, normalizedServer, req.Token, req.SeedBindings); err != nil {
		return nil, err
	}

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

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
		return nil, err
	}

	return finishPullRuns(t, req, resp, serverURL, agentHome, false, agentBin)
}

func finishPullRuns(t *testing.T, req *Request, resp *Response, serverURL, agentHome string, inProcess bool, agentBin string) (*Response, error) {
	t.Helper()

	if len(req.Args) == 0 {
		return nil, fmt.Errorf("Request.Args is required")
	}

	invocations := 1
	if req.WorktreeCollision {
		invocations = 2
	}

	var lastStdout, lastStderr string
	var lastExit int

	for i := 0; i < invocations; i++ {
		if req.WorktreeCollision && i == 1 {
			if err := reDirtyTopLevel(t, resp.ProjectDir); err != nil {
				return nil, err
			}
		}

		full := []string{"--server", serverURL, "--token", req.Token}
		full = append(full, req.Args...)
		mode := "L2-inprocess"
		if !inProcess {
			mode = "L3-binary"
		}
		t.Logf("%s remote-agent argv: %v", mode, full)

		var exitCode int
		var out, errOut string
		var runErr error
		if inProcess {
			exitCode, out, errOut, runErr = runAgentInProcess(full, agentHome, req.PipeStdin)
		} else {
			exitCode, out, errOut, runErr = runAgentBinary(agentBin, full, agentHome, req.PipeStdin)
		}
		if runErr != nil {
			return nil, runErr
		}
		lastExit = exitCode
		lastStdout = out
		lastStderr = errOut
		resp.InvocationCount++

		if req.WorktreeCollision && i == 0 && lastExit != 0 {
			return nil, fmt.Errorf("first pull-local failed exit %d:\n%s\n%s", lastExit, out, errOut)
		}
	}

	resp.ExitCode = lastExit
	resp.Stdout = lastStdout
	resp.Stderr = lastStderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	return resp, nil
}

func runAgentInProcess(argv []string, agentHome string, pipeStdin bool) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	testhooks.SetHomeOverride(agentHome)
	defer testhooks.ResetInProcessOverrides()

	oldIn := os.Stdin
	var stdinFile *os.File
	if pipeStdin {
		r, w, err := os.Pipe()
		if err != nil {
			return 0, "", "", err
		}
		_ = w.Close()
		stdinFile = r
		os.Stdin = r
	} else {
		devNull, err := os.Open(os.DevNull)
		if err != nil {
			return 0, "", "", err
		}
		stdinFile = devNull
		os.Stdin = devNull
	}
	defer func() {
		os.Stdin = oldIn
		_ = stdinFile.Close()
	}()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return 0, "", "", err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		stdoutR.Close()
		stdoutW.Close()
		return 0, "", "", err
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

	stdout := string(outBytes)
	stderr := string(errBytes)
	exitCode := 0
	if runErr != nil {
		stderr += fmt.Sprintf("Error: %v\n", runErr)
		exitCode = 1
	}
	return exitCode, stdout, stderr, nil
}

func runAgentBinary(bin string, argv []string, agentHome string, pipeStdin bool) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = stripEnvPrefix(agentEnv, "GIT_CONFIG_COUNT=")
	agentEnv = stripEnvPrefix(agentEnv, "GIT_CONFIG_KEY_")
	agentEnv = stripEnvPrefix(agentEnv, "GIT_CONFIG_VALUE_")
	agentEnv = append(agentEnv, "HOME="+agentHome)
	agentEnv = append(agentEnv,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=protocol.file.allow",
		"GIT_CONFIG_VALUE_0=always",
	)
	cmd.Env = agentEnv
	if pipeStdin {
		cmd.Stdin = strings.NewReader("")
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, "", "", runErr
		}
	}
	return exitCode, outBuf.String(), errBuf.String(), nil
}

func writeProjectsJSON(configHome string, projectsList []ProjectEntry) error {
	rows := make([]projectsFileRow, 0, len(projectsList))
	for _, project := range projectsList {
		if project.ID == "" || project.Name == "" || project.Dir == "" {
			return fmt.Errorf("project id, name, and dir are required")
		}
		rows = append(rows, projectsFileRow{
			ID:              project.ID,
			Name:            project.Name,
			RepoURL:         "",
			Dir:             project.Dir,
			GitUserConfigID: project.GitUserConfigID,
			GitUserName:     project.GitUserName,
			GitUserEmail:    project.GitUserEmail,
			CreatedAt:       "2026-06-29T00:00:00Z",
		})
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configHome, "projects.json"), data, 0644)
}

func writeRemoteAgentConfig(path, server, token string, bindings []ProjectBinding) error {
	cfg := remoteAgentConfigFile{
		Default: server,
		Domains: []domainConfigRow{{Server: server, Token: token}},
	}
	for _, b := range bindings {
		cfg.ProjectBindings = append(cfg.ProjectBindings, bindingRow{
			Server:    strings.TrimRight(strings.TrimSpace(b.Server), "/"),
			RemoteDir: b.RemoteDir,
			LocalPath: b.LocalPath,
		})
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func reDirtyTopLevel(t *testing.T, projectDir string) error {
	t.Helper()
	p := filepath.Join(projectDir, "after-first-pull.txt")
	return os.WriteFile(p, []byte("dirty again\n"), 0644)
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

func gitPorcelain(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git status --porcelain in %s: %v\n%s", dir, err, out)
	}
	return string(out)
}

func readConfigBindings(t *testing.T, path string) []bindingRow {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg remoteAgentConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return cfg.ProjectBindings
}

func worktreeBaseDir(agentHome string) string {
	return filepath.Join(agentHome, ".ai-critic", "remote-agent", "project-worktrees")
}

func portBaseFromTestName(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return 25000 + (hash % 1000)
}

func pickFreePort(base int) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		port := ln.Addr().(*net.TCPAddr).Port
		ln.Close()
		return port
	}
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	panic(fmt.Sprintf("no free port near %d", base))
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
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
