# Remote-Agent Project List Git Status Doctests

End-to-end tests for `remote-agent project list`: live Git branch, commit, and
worktree cleanliness rendered from server-side inspection of each project's `dir`.

# DSN (Domain Specific Notion)

The harness exercises the remote profile of the shared agent CLI against a real
`ai-critic-server` subprocess with seeded `projects.json` and temp git worktrees.

**Participants**

- **remote-agent subprocess** — built from `./cmd/remote-agent`; calls
  `GET /api/projects?all=true` via `project list`.
- **ai-critic-server subprocess** — bound to an ephemeral port with test
  credentials; reads `projects.json` from isolated `AI_CRITIC_HOME`.
- **Temp project directories** — leaf `Setup` creates git repos (or plain dirs)
  and registers them in `projects.json` before the CLI runs.
- **Test credentials** — `lib.TestPassword` token written to `server-credentials`.
- **Isolated agent HOME** — temp `HOME` with `~/.ai-critic/remote-agent-config.json`
  (`project_bindings`) so list output never reads the developer machine config.
- **Local path bindings** — optional `(server, remote_dir) → local_path` rows resolved
  when `printProjectGitConfig` renders each project.

**Behaviors**

- `Local Dir:` appears immediately after `Dir:` — bound absolute path or `-` when no match.
- Binding lookup uses normalized `--server` URL and API `project.Dir` (same as bind-local).
- Clean repos show branch name, 7-char commit hash + subject, `Worktree: clean`.
- Dirty repos show per-type counts: added (includes untracked), changed, renamed, deleted.
- Detached HEAD shows `Git Branch: (detached)` with commit still populated.
- Non-git directories show `-` for branch, commit, and worktree lines.
- Git identity fields remain after the new git status lines.
- `--dirty` requests `GET /api/projects?all=true&dirty=true`; server omits clean and non-git projects.
- Auth failures from `project list` include a friendly `remote-agent config` hint
  without local-only credential-file guidance.
- `remote-agent auth import-local` is rejected as local-agent-only and does not
  read local server credentials or mutate `remote-agent-config.json`.

## Version

0.0.2

## Decision Tree

```
[remote-agent project list — git status]
 |
 +-- clean-repo/                    (LEAF)  clean repo → branch, commit, clean worktree
 +-- dirty-repo/                    (LEAF)  mixed porcelain → dirty counts
 +-- detached-head/                 (LEAF)  detached HEAD → (detached)
 +-- not-git-repo/                  (LEAF)  plain dir → dashes
 +-- identity-fields-preserved/     (LEAF)  identity lines + new git lines
 +-- list-dirty/
 |    +-- shows-dirty-only/         (LEAF)  --dirty omits clean project
 |    +-- empty-all-clean/          (LEAF)  --dirty with only clean → no dirty message
 |
 +-- auth-failure/
 |    +-- bad-token-guidance/       (LEAF)  project list bad token → remote-agent config hint only
 |
 +-- local-only-auth-helper/
 |    +-- rejected-by-remote/       (LEAF)  auth import-local rejected and config untouched
 |
 +-- local-dir/                      (GROUP)  CLI binding resolution for Local Dir line
      +-- bound/                     (LEAF)  seeded binding → absolute Local Dir
      +-- unbound/                   (LEAF)  no binding → Local Dir: -
      +-- bound-dirty-filter/        (LEAF)  --dirty + binding on dirty project
      +-- wrong-server/              (LEAF)  binding for other server → -
      +-- wrong-remote-dir/          (LEAF)  binding for other remote_dir → -
      +-- git-config-get-bound/      (LEAF)  git-config get shows bound Local Dir
      +-- git-config-get-unbound/    (LEAF)  git-config get shows Local Dir: -
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `clean-repo` | Initial commit on `main`; branch, short hash + message, clean worktree |
| 2 | `dirty-repo` | Untracked, modified, renamed, deleted → dirty counts |
| 3 | `detached-head` | Detached checkout; branch `(detached)`, commit still shown |
| 4 | `not-git-repo` | Directory without `.git`; git lines show `-` |
| 5 | `identity-fields-preserved` | Saved git identity fields unchanged alongside git status |
| 6 | `list-dirty/shows-dirty-only` | Two projects; `--dirty` prints only the dirty one |
| 7 | `list-dirty/empty-all-clean` | One clean project; `--dirty` → `No dirty projects found.` |
| 8 | `auth-failure/bad-token-guidance` | `project list` bad token prints remote auth guidance |
| 9 | `local-only-auth-helper/rejected-by-remote` | `auth import-local` rejected for remote-agent |
| 10 | `local-dir/bound` | Seeded binding; `Local Dir` shows absolute path after `Dir` |
| 11 | `local-dir/unbound` | Isolated empty config; `Local Dir: -` |
| 12 | `local-dir/bound-dirty-filter` | `--dirty` lists bound dirty project with Local Dir |
| 13 | `local-dir/wrong-server` | Binding server mismatch → dash |
| 14 | `local-dir/wrong-remote-dir` | Binding remote_dir mismatch → dash |
| 15 | `local-dir/git-config-get-bound` | `project git-config get` includes bound Local Dir |
| 16 | `local-dir/git-config-get-unbound` | `project git-config get` with no binding → dash |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Clean worktree | clean-repo, identity-fields-preserved |
| Dirty worktree (all four types) | dirty-repo |
| Detached HEAD | detached-head |
| Not a git repo | not-git-repo |
| Git identity metadata | identity-fields-preserved |
| `--dirty` filter | list-dirty/*, local-dir/bound-dirty-filter |
| Auth failure messaging | auth-failure/bad-token-guidance |
| Local-only helper rejection | local-only-auth-helper/rejected-by-remote |
| Local binding present | local-dir/bound, bound-dirty-filter, git-config-get-bound |
| Local binding absent | local-dir/unbound, git-config-get-unbound, legacy leaves |
| Binding key mismatch | wrong-server, wrong-remote-dir |
| Subcommand `project list` vs `git-config get` | local-dir/* |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-project-list
doctest test ./tests/remote-agent-project-list/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
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

	SeedBindings []ProjectBinding
	LocalPath    string
	WatchRemoteConfig bool
	ServerCredentialContent string
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
	RemoteConfigBefore []byte
	RemoteConfigAfter  []byte
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

	if len(req.Args) == 0 {
		req.Args = []string{"project", "list"}
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-server-project-list-"+safeName)
	agentBin := filepath.Join(os.TempDir(), "remote-agent-project-list-"+safeName)

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

	agentHome, err := os.MkdirTemp("", "remote-agent-list-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome
	aiCriticAgent := filepath.Join(agentHome, ".ai-critic")
	if err := os.MkdirAll(aiCriticAgent, 0755); err != nil {
		return nil, err
	}

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

		if err := writeProjectsJSON(configHome, projects); err != nil {
			return nil, err
		}
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

	configPath := filepath.Join(aiCriticAgent, "remote-agent-config.json")
	if err := writeRemoteAgentConfig(configPath, normalizedServer, req.Token, req.SeedBindings); err != nil {
		return nil, err
	}
	resp.RemoteConfigPath = configPath
	if req.ServerCredentialContent != "" {
		credPath := filepath.Join(aiCriticAgent, "server-credentials")
		if err := os.WriteFile(credPath, []byte(req.ServerCredentialContent), 0600); err != nil {
			return nil, err
		}
	}
	if req.WatchRemoteConfig {
		resp.RemoteConfigBefore, _ = os.ReadFile(resp.RemoteConfigPath)
	}

	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)
	t.Logf("remote-agent argv: %v", argv)

	agentCmd := exec.Command(agentBin, argv...)
	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)
	agentCmd.Env = agentEnv

	var stdout, stderr bytes.Buffer
	agentCmd.Stdout = &stdout
	agentCmd.Stderr = &stderr

	runErr := agentCmd.Run()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			return nil, runErr
		}
	}
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	if req.WatchRemoteConfig {
		resp.RemoteConfigAfter, _ = os.ReadFile(resp.RemoteConfigPath)
	}

	return resp, nil
}

func writeProjectsJSON(configHome string, projects []ProjectEntry) error {
	rows := make([]projectsFileRow, 0, len(projects))
	for _, project := range projects {
		if project.ID == "" {
			return fmt.Errorf("project ID is required")
		}
		if project.Name == "" {
			return fmt.Errorf("project name is required")
		}
		if project.Dir == "" {
			return fmt.Errorf("project dir is required")
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
	path := filepath.Join(configHome, "projects.json")
	return os.WriteFile(path, data, 0644)
}

func writeRemoteAgentConfig(path, server, token string, bindings []ProjectBinding) error {
	cfg := remoteAgentConfigFile{
		Default: server,
		Domains: []domainConfigRow{{Server: server, Token: token}},
	}
	for _, b := range bindings {
		srv := strings.TrimRight(strings.TrimSpace(b.Server), "/")
		if srv == "" {
			srv = server
		}
		cfg.ProjectBindings = append(cfg.ProjectBindings, bindingRow{
			Server:    srv,
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

// portBaseFromTestName maps each parallel leaf package to a distinct starting
// port in [25000, 25999] so concurrent doctest runs do not all bind 24800.
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
