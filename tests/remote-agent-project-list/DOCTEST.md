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

**Behaviors**

- Clean repos show branch name, 7-char commit hash + subject, `Worktree: clean`.
- Dirty repos show per-type counts: added (includes untracked), changed, renamed, deleted.
- Detached HEAD shows `Git Branch: (detached)` with commit still populated.
- Non-git directories show `-` for branch, commit, and worktree lines.
- Git identity fields remain after the new git status lines.
- `--dirty` requests `GET /api/projects?all=true&dirty=true`; server omits clean and non-git projects.

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
      +-- shows-dirty-only/         (LEAF)  --dirty omits clean project
      +-- empty-all-clean/          (LEAF)  --dirty with only clean → no dirty message
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

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Clean worktree | clean-repo, identity-fields-preserved |
| Dirty worktree (all four types) | dirty-repo |
| Detached HEAD | detached-head |
| Not a git repo | not-git-repo |
| Git identity metadata | identity-fields-preserved |
| `--dirty` filter | list-dirty/* |

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

type Request struct {
	Args   []string
	Server string
	Token  string

	Project  ProjectEntry
	Projects []ProjectEntry
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ConfigHome string
	ProjectDir string
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

	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)
	t.Logf("remote-agent argv: %v", argv)

	agentCmd := exec.Command(agentBin, argv...)
	agentCmd.Env = os.Environ()

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