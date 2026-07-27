# Remote-Agent Project List Git Status Doctests

Doctests for `remote-agent project list`: live Git branch, commit, and worktree
cleanliness rendered from server-side inspection of each project's `dir`.

Most leaves are **L2 in-process** (`projects.RegisterAPIForFile` + `agentcli.Run`
with testhooks home override). One sparse **L3 e2e** smoke keeps the product binary path.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `projects.RegisterAPIForFile` on a local mux +
local bearer auth + `agentcli.Run` with mutex-scoped home override (no product
binaries, no process Setenv). One sparse **L3 e2e** smoke keeps the binary path
(`UseCLI` + `label: heavy, e2e`): `clean-repo`.

**Participants**

- **L2: agentcli.Run** — in-process `project list` / `git-config get` against the local mux.
- **L2: projects HTTP** — `RegisterAPIForFile(mux, configHome/projects.json)` on ephemeral port.
- **L3: remote-agent + ai-critic-server subprocesses** — sparse `UseCLI` smoke.
- **Temp project directories** — leaf `Setup` creates git repos (or plain dirs)
  and registers them in `projects.json` before the CLI runs.
- **Test credentials** — `lib.TestPassword` token for bearer auth / L3 credentials file.
- **Isolated agent HOME** — temp dir with `remote-agent-config.json` (`project_bindings`);
  L2 uses `testhooks.SetHomeOverride`, L3 uses child `HOME=`.
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

0.0.3

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
doctest test ./tests/remote-agent-project-list/...          # unlabeled L2 mass
doctest test --label e2e ./tests/remote-agent-project-list/...  # ~1 L3 smoke
```

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
	"github.com/xhd2015/ai-critic/server/projects"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run (profile + stdout/stderr + home override).
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

	SeedBindings []ProjectBinding
	LocalPath    string
	WatchRemoteConfig bool
	ServerCredentialContent string

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

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		req.Args = []string{"project", "list"}
	}
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
		if err := writeProjectsJSON(configHome, projectsList); err != nil {
			return nil, err
		}
	}

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, moduleRoot, cacheDir, configHome, agentHome, aiCriticAgent)
	}
	return runInProcessL2(t, d, req, resp, configHome, agentHome, aiCriticAgent)
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

func runInProcessL2(t *testing.T, d *session.Doctest, req *Request, resp *Response, configHome, agentHome, aiCriticAgent string) (*Response, error) {
	t.Helper()

	mux := http.NewServeMux()
	projects.RegisterAPIForFile(mux, filepath.Join(configHome, "projects.json"))
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := withBearerAuth(lib.TestPassword, mux)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen in-process project-list server: %w", err)
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

	if err := prepareAgentConfig(t, req, resp, aiCriticAgent, normalizedServer); err != nil {
		return nil, err
	}

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 10*time.Second); err != nil {
		return nil, err
	}
	t.Logf("L2 in-process project-list server on %s", normalizedServer)

	return finishListRun(t, req, resp, serverURL, agentHome, true)
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, moduleRoot, cacheDir, configHome, agentHome, aiCriticAgent string) (*Response, error) {
	t.Helper()

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

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	if err := prepareAgentConfig(t, req, resp, aiCriticAgent, normalizedServer); err != nil {
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

	return finishListRunBinary(t, req, resp, serverURL, agentBin, agentHome)
}

func prepareAgentConfig(t *testing.T, req *Request, resp *Response, aiCriticAgent, normalizedServer string) error {
	t.Helper()
	if req.LocalPath != "" {
		absLocal, err := filepath.Abs(req.LocalPath)
		if err != nil {
			return err
		}
		req.LocalPath = absLocal
		resp.LocalPath = absLocal
	}
	configPath := filepath.Join(aiCriticAgent, "remote-agent-config.json")
	if err := writeRemoteAgentConfig(configPath, normalizedServer, req.Token, req.SeedBindings); err != nil {
		return err
	}
	resp.RemoteConfigPath = configPath
	if req.ServerCredentialContent != "" {
		credPath := filepath.Join(aiCriticAgent, "server-credentials")
		if err := os.WriteFile(credPath, []byte(req.ServerCredentialContent), 0600); err != nil {
			return err
		}
	}
	if req.WatchRemoteConfig {
		resp.RemoteConfigBefore, _ = os.ReadFile(resp.RemoteConfigPath)
	}
	return nil
}

func finishListRun(t *testing.T, req *Request, resp *Response, serverURL, agentHome string, inProcess bool) (*Response, error) {
	t.Helper()
	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)
	t.Logf("L2-inprocess remote-agent argv: %v", argv)

	exitCode, stdout, stderr, runErr := runAgentInProcess(argv, agentHome)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	if req.WatchRemoteConfig {
		resp.RemoteConfigAfter, _ = os.ReadFile(resp.RemoteConfigPath)
	}
	return resp, nil
}

func finishListRunBinary(t *testing.T, req *Request, resp *Response, serverURL, agentBin, agentHome string) (*Response, error) {
	t.Helper()
	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)
	t.Logf("L3-binary remote-agent argv: %v", argv)

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

func runAgentInProcess(argv []string, agentHome string) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	testhooks.SetHomeOverride(agentHome)
	defer testhooks.ResetInProcessOverrides()

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

func writeProjectsJSON(configHome string, projectsList []ProjectEntry) error {
	rows := make([]projectsFileRow, 0, len(projectsList))
	for _, project := range projectsList {
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
