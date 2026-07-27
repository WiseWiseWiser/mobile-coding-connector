# Remote-Agent Git Local Commands Doctests

Doctests for `remote-agent git -C <dir> <subcommand> [args...]`: allowlisted
read-only (or local-only) git subcommands run via `POST /api/remote-agent/git/run`,
streaming stdout/stderr and mirroring the remote exit code.

Most leaves are **L2 in-process** (`server/git.RegisterAPI` + `agentcli.Run`).
Two sparse **L3 e2e** smokes keep the product binary path.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `server/git.RegisterAPI` on a local mux +
`agentcli.Run` (no product binaries). Two sparse **L3 e2e** smokes keep the binary
path (`UseCLI` + `label: heavy, e2e`): `allowlisted/status/clean-repo` and
`server-rejected/denied-mutating/add`.

**L2** serves git run/clone/fetch/pull/push APIs in-process via `git.RegisterAPI(mux)`.
**L3 smokes** still run product binaries with `AI_CRITIC_HOME=configHome`.

**Participants**

- **L2: agentcli.Run** — in-process `git -C` CLI against the local mux
  (stdout/stderr captured; serialized with a process mutex).
- **L2: git HTTP** — `RegisterAPI` on ephemeral port; same wire path
  (`POST /api/remote-agent/git/run` NDJSON stream).
- **L3: remote-agent + ai-critic-server subprocesses** — sparse `UseCLI` smokes.
- **Temp repository directories** — leaf `Setup` creates git repos (or plain dirs)
  on disk before the CLI runs; path is passed verbatim in `-C`.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/remote-agent-git-local-doctest-<id>/` for L3 shared binaries (file lock).

**Behaviors**

- Allowlisted local commands (`status`, `diff`, `log`, `branch`, `rev-parse`,
  `show`, read-only `remote`/`config`, `stash list`/`show`) stream git output and
  exit 0 on success.
- Dedicated network ops (`clone`, `fetch`, `pull`, `push`) and mutating git
  subcommands are rejected before spawning git (CLI and/or server allowlist).
- Missing `-C` or unknown CLI subcommands fail locally without calling `/run`.
- Non-repository `dir` values fail server validation with a clear message.

## Version

0.0.3

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
 |         +-- add/                       (LEAF)  git add [L3 smoke]
 |         +-- config-set/                (LEAF)  git config --set
 |         +-- branch-delete/             (LEAF)  git branch -D
 |         +-- remote-add/                (LEAF)  git remote add
 |
 +-- allowlisted/                          (GROUP) /run succeeds
      +-- status/
      |    +-- clean-repo/                (LEAF) [L3 smoke]
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
| 4 | `server-rejected/denied-mutating/add` | `git add` blocked (L3 smoke) |
| 5 | `server-rejected/denied-mutating/config-set` | `git config --set` blocked |
| 6 | `server-rejected/denied-mutating/branch-delete` | `git branch -D` blocked |
| 7 | `server-rejected/denied-mutating/remote-add` | `git remote add` blocked |
| 8 | `allowlisted/status/clean-repo` | Clean worktree on `main` (L3 smoke) |
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
doctest test ./tests/remote-agent-git-local/...          # unlabeled L2 mass
doctest test --label e2e ./tests/remote-agent-git-local/...  # ~2 L3 smokes
```

```go
import (
	"bytes"
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
	"github.com/xhd2015/ai-critic/script/lib"
	servergit "github.com/xhd2015/ai-critic/server/git"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run (package-level active
// profile + temporary stdout/stderr swaps are process-global).
var agentcliInProcessMu sync.Mutex

type Request struct {
	Args   []string
	Server string
	Token  string

	// RepoDir is the absolute path used with -C (set by leaf Setup).
	RepoDir string

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
	RepoDir    string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		return nil, fmt.Errorf("Request.Args is required (e.g. git -C <dir> status)")
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	// DOCTEST_ROOT is tests/remote-agent-git-local; module root is two levels up.
	// Do not walk from cwd: doctest runs under mapping-gen which has its own go.mod.
	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	if req.RepoDir != "" {
		absDir, err := filepath.Abs(req.RepoDir)
		if err != nil {
			return nil, fmt.Errorf("abs repo dir: %w", err)
		}
		req.RepoDir = absDir
		resp.RepoDir = absDir
	}

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, moduleRoot, cacheDir, configHome)
	}
	return runInProcessL2(t, d, req, resp)
}

func runInProcessL2(t *testing.T, d *session.Doctest, req *Request, resp *Response) (*Response, error) {
	t.Helper()

	mux := http.NewServeMux()
	servergit.RegisterAPI(mux)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen in-process git server: %w", err)
	}
	serverPort := ln.Addr().(*net.TCPAddr).Port
	resp.ServerPort = serverPort
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://127.0.0.1:%d", serverPort)
	}

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 10*time.Second); err != nil {
		return nil, err
	}
	t.Logf("L2 in-process git server on %s", serverURL)

	return finishGitRun(t, req, resp, serverURL, func(argv []string) (int, string, string, error) {
		return runAgentInProcess(argv)
	}, true)
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, moduleRoot, cacheDir, configHome string) (*Response, error) {
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

	return finishGitRun(t, req, resp, serverURL, func(argv []string) (int, string, string, error) {
		return runAgentBinary(agentBin, argv)
	}, false)
}

func finishGitRun(
	t *testing.T,
	req *Request,
	resp *Response,
	serverURL string,
	runCLI func(argv []string) (int, string, string, error),
	inProcess bool,
) (*Response, error) {
	t.Helper()

	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)

	mode := "L2-inprocess"
	if !inProcess {
		mode = "L3-binary"
	}
	t.Logf("%s remote-agent argv: %v", mode, argv)

	exitCode, stdout, stderr, runErr := runCLI(argv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	return resp, nil
}

func runAgentInProcess(argv []string) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

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

func runAgentBinary(bin string, argv []string) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	cmd.Env = os.Environ()
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
