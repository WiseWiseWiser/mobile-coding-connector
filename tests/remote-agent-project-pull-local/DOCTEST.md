# Remote-Agent Project bind-local & pull-local Doctests

End-to-end tests for `remote-agent project bind-local` and
`remote-agent project pull-local`: origin validation, config bindings, dirty-state
transfer into local git worktrees, flags, and submodule guards.

# DSN (Domain Specific Notion)

The harness exercises the remote profile against a live `ai-critic-server` with
registered project directories (temp git repos on the test machine). For
`pull-local`, the server exposes `POST /api/remote-agent/project/pull-local`
(dry-run JSON plan or streamed tar.gz package) and
`POST /api/remote-agent/project/pull-local/truncate`; the CLI applies the
package locally. `bind-local` remains local-config only. Local git operations run
in an isolated `HOME` so `remote-agent-config.json` and `project-worktrees/`
never touch the developer machine.

**Participants**

- **remote-agent subprocess** — `./cmd/remote-agent`; subcommands `project bind-local`
  and `project pull-local` with `--server` / `--token`.
- **ai-critic-server subprocess** — ephemeral port, `AI_CRITIC_HOME` with
  `projects.json` and credentials; enforces pull-local guards (clean/dirty,
  submodules, per-file 1MB cap, total package cap) and builds tar.gz payloads.
- **Remote project directory** — temp clone registered as `ProjectInfo.Dir`; may
  include a clean or dirty submodule tree.
- **Local git repository** — separate temp clone sharing a `file://` bare origin
  with the remote project repo.
- **Isolated agent HOME** — temp `HOME` → `~/.ai-critic/remote-agent-config.json`
  (`project_bindings`) and `~/.ai-critic/remote-agent/project-worktrees/…`.

**Behaviors**

- `bind-local` upserts `(server, remote_dir) → local_path` after same-origin check.
- `pull-local` refuses clean remotes, missing bindings on non-TTY stdin, origin
  mismatch, and dirty submodules (including before `--dry-run` plan output).
- Successful pull calls the server package endpoint, creates a detached worktree
  at the computed slug path, applies `patch.diff` and untracked members from the
  tarball, and by default truncates the remote repo via the truncate API.
- Per-file and total byte limits reject oversized pulls unless
  `--include-file` or `--max-size` overrides apply.
- `--no-truncate-remote` leaves remote porcelain intact; `--dry-run` prints a plan
  without worktree or remote mutations.
- Repeated pulls on the same branch allocate incrementing worktree suffixes (`main-1`,
  `main-2`, …).

## Version

0.0.2

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
      +-- worktree-collision/            (LEAF)   two pulls → main-2 when main-1 exists
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
| 9 | `pull-local/worktree-collision` | Second pull uses `main-2` suffix |
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
| Submodule clean vs dirty | submodule-clean, submodule-dirty, dry-run-submodule-dirty |
| Per-file 1MB cap | oversized-file-rejected, include-file-allows-large |
| `--include-file` valid vs invalid | include-file-allows-large, include-file-not-dirty |
| Total 64MB cap vs `--max-size` | total-over-max, max-size-override |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-project-pull-local
doctest test -v --gen-dir /tmp/pull-local-server ./tests/remote-agent-project-pull-local/...
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
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

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

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot := findModuleRoot()

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-server-pull-local-"+safeName)
	agentBin := filepath.Join(os.TempDir(), "remote-agent-pull-local-"+safeName)

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
			return nil, fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
		}
		t.Cleanup(func() { os.Remove(spec.out) })
	}

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

	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort := pickFreePort(portBase)
	resp.ServerPort = serverPort

	projects := req.Projects
	if len(projects) == 0 && req.Project.Dir != "" {
		projects = []ProjectEntry{req.Project}
	}
	if len(projects) > 0 {
		for i := range projects {
			absDir, err := filepath.Abs(projects[i].Dir)
			if err != nil {
				return nil, fmt.Errorf("abs project dir: %w", err)
			}
			projects[i].Dir = absDir
		}
		req.Projects = projects
		resp.ProjectDir = projects[0].Dir
		req.RemoteDirAfterSetup = projects[0].Dir
	}

	if len(projects) > 0 {
		if err := writeProjectsJSON(configHome, projects); err != nil {
			return nil, err
		}
	}

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	if req.LocalPath != "" {
		absLocal, err := filepath.Abs(req.LocalPath)
		if err != nil {
			return nil, err
		}
		req.LocalPath = absLocal
		resp.LocalPath = absLocal
	}

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

	invocations := 1
	if req.WorktreeCollision {
		invocations = 2
	}

	argv := req.Args
	if len(argv) == 0 {
		return nil, fmt.Errorf("Request.Args is required")
	}

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

	var lastStdout, lastStderr bytes.Buffer
	var lastExit int

	for i := 0; i < invocations; i++ {
		if req.WorktreeCollision && i == 1 {
			if err := reDirtyTopLevel(t, resp.ProjectDir); err != nil {
				return nil, err
			}
		}

		full := []string{"--server", serverURL, "--token", req.Token}
		full = append(full, argv...)
		t.Logf("remote-agent argv: %v", full)

		agentCmd := exec.Command(agentBin, full...)
		agentCmd.Env = agentEnv
		if req.PipeStdin {
			agentCmd.Stdin = strings.NewReader("")
		}

		var outBuf, errBuf bytes.Buffer
		agentCmd.Stdout = &outBuf
		agentCmd.Stderr = &errBuf

		runErr := agentCmd.Run()
		if runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				lastExit = exitErr.ExitCode()
			} else {
				return nil, runErr
			}
		}
		lastStdout = outBuf
		lastStderr = errBuf
		resp.InvocationCount++

		if req.WorktreeCollision && i == 0 && lastExit != 0 {
			return nil, fmt.Errorf("first pull-local failed exit %d:\n%s\n%s", lastExit, outBuf.String(), errBuf.String())
		}
	}

	resp.ExitCode = lastExit
	resp.Stdout = lastStdout.String()
	resp.Stderr = lastStderr.String()
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	return resp, nil
}

func writeProjectsJSON(configHome string, projects []ProjectEntry) error {
	rows := make([]projectsFileRow, 0, len(projects))
	for _, project := range projects {
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

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found")
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
	return 25000 + (hash % 1000)
}

func pickFreePort(base int) int {
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