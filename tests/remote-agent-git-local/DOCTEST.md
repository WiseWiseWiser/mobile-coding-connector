# Remote-Agent Git Local Commands Doctests

End-to-end tests for `remote-agent git -C <dir> <subcommand> [args...]`:
allowlisted read-only (or local-only) git subcommands run on `ai-critic-server` via
`POST /api/remote-agent/git/run`, streaming stdout/stderr and mirroring the remote
exit code.

# DSN (Domain Specific Notion)

The harness exercises the remote profile of the shared agent CLI against a real
`ai-critic-server` subprocess and temp git worktrees on the same host (the server's
`dir` is an absolute path to those trees).

**Participants**

- **remote-agent subprocess** — built from `./cmd/remote-agent`; parses `git -C <dir>`
  and dispatches allowlisted subcommands to the client.
- **HTTP client** — `POST /api/remote-agent/git/run` with JSON `{dir, args, ...}`.
- **ai-critic-server subprocess** — validates `dir`, allowlists `args[0]`, spawns
  `git` with NDJSON streaming (`stdout`, `stderr`, `heartbeat`, `exit`, `error`).
- **Temp repository directories** — leaf `Setup` creates git repos (or plain dirs)
  on disk before the CLI runs; path is passed verbatim in `-C`.

**Behaviors**

- Allowlisted local commands (`status`, `diff`, `log`, `branch`, `rev-parse`,
  `show`, read-only `remote`/`config`, `stash list`/`show`) stream git output and
  exit 0 on success.
- Dedicated network ops (`clone`, `fetch`, `pull`, `push`) and mutating git
  subcommands are rejected before spawning git (CLI and/or server allowlist).
- Missing `-C` or unknown CLI subcommands fail locally without calling `/run`.
- Non-repository `dir` values fail server validation with a clear message.

## Version

0.0.2

## Decision Tree

```
[remote-agent git -C local commands]
 |
 +-- cli-rejected/                         (GROUP) validation before HTTP
 |    +-- missing-c-dir/                  (LEAF)  git status without -C
 |    +-- unknown-subcommand/             (LEAF)  frobnicate with valid repo
 |
 +-- server-rejected/                      (GROUP) HTTP/API gate before git spawn
 |    +-- not-git-repo/                   (LEAF)  plain directory
 |    +-- denied-mutating/                 (GROUP) allowlist denies mutation
 |         +-- add/                       (LEAF)  git add
 |         +-- config-set/                (LEAF)  git config --set
 |         +-- branch-delete/             (LEAF)  git branch -D
 |         +-- remote-add/                (LEAF)  git remote add
 |
 +-- allowlisted/                          (GROUP) /run succeeds
      +-- status/
      |    +-- clean-repo/                (LEAF)
      |    +-- dirty-repo/                (LEAF)
      +-- diff/
      |    +-- unstaged-hunk/             (LEAF)
      |    +-- cached/                    (LEAF)
      +-- log/
      |    +-- oneline-two-commits/       (LEAF)
      +-- branch/
      |    +-- lists-current/             (LEAF)
      +-- rev-parse/
      |    +-- head/                      (LEAF)
      +-- show/
      |    +-- latest-commit/             (LEAF)
      +-- remote/
      |    +-- list-verbose/              (LEAF)
      +-- config/
      |    +-- get-user-name/             (LEAF)
      +-- stash/
           +-- list-empty/                (LEAF)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `cli-rejected/missing-c-dir` | `git status` without `-C` → CLI requires `-C` |
| 2 | `cli-rejected/unknown-subcommand` | `frobnicate` → unknown subcommand before HTTP |
| 3 | `server-rejected/not-git-repo` | Plain dir → not a git repository |
| 4 | `server-rejected/denied-mutating/add` | `git add` blocked |
| 5 | `server-rejected/denied-mutating/config-set` | `git config --set` blocked |
| 6 | `server-rejected/denied-mutating/branch-delete` | `git branch -D` blocked |
| 7 | `server-rejected/denied-mutating/remote-add` | `git remote add` blocked |
| 8 | `allowlisted/status/clean-repo` | Clean worktree on `main` |
| 9 | `allowlisted/status/dirty-repo` | Modified + untracked paths in status |
| 10 | `allowlisted/diff/unstaged-hunk` | Working tree diff hunk |
| 11 | `allowlisted/diff/cached` | `diff --cached` shows staged change |
| 12 | `allowlisted/log/oneline-two-commits` | Two `--oneline` lines, newest first |
| 13 | `allowlisted/branch/lists-current` | Current branch marked `* main` |
| 14 | `allowlisted/rev-parse/head` | `rev-parse HEAD` → commit hash |
| 15 | `allowlisted/show/latest-commit` | `show` includes commit subject |
| 16 | `allowlisted/remote/list-verbose` | `remote -v` lists configured remote |
| 17 | `allowlisted/config/get-user-name` | `config --get user.name` |
| 18 | `allowlisted/stash/list-empty` | `stash list` exit 0, empty output |

## Parameter Coverage

| Factor (significance →) | Leaves |
|-------------------------|--------|
| Failure phase (CLI vs server vs success) | cli-rejected/*, server-rejected/*, allowlisted/* |
| Allowlisted subcommand | allowlisted/* |
| Worktree state | status/clean-repo, status/dirty-repo, diff/* |
| Git passthrough args | diff/cached, log/oneline-two-commits, config/get-user-name |
| Denied mutation class | denied-mutating/* |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-git-local
doctest test ./tests/remote-agent-git-local/...
```

```go
import (
	"bytes"
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

type Request struct {
	Args   []string
	Server string
	Token  string

	// RepoDir is the absolute path used with -C (set by leaf Setup).
	RepoDir string
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ConfigHome string
	RepoDir    string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		return nil, fmt.Errorf("Request.Args is required (e.g. git -C <dir> status)")
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	serverBin := filepath.Join(os.TempDir(), "ai-critic-server-git-local-"+safeName)
	agentBin := filepath.Join(os.TempDir(), "remote-agent-git-local-"+safeName)

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

	if req.RepoDir != "" {
		absDir, err := filepath.Abs(req.RepoDir)
		if err != nil {
			return nil, fmt.Errorf("abs repo dir: %w", err)
		}
		req.RepoDir = absDir
		resp.RepoDir = absDir
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