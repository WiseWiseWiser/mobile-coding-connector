# Remote-Agent Download Directory Doctests

End-to-end tests for `remote-agent download <REMOTE_PATH> [LOCAL_PATH]` when the
remote source is a file or directory. Directory downloads mirror a remote tree
locally via client-orchestrated per-file GET downloads with streaming progress,
transient retry, and filesystem-based resume (skip complete files, Range for
partial files).

# DSN (Domain Specific Notion)

The harness exercises the remote-agent CLI against a real `ai-critic-server`
subprocess. The server runs with `HOME=serverHome` so remote paths resolve under
an isolated fake machine home. The CLI runs in a separate `agentHome` with only
`remote-agent-config.json`, downloading into a per-leaf `agentWorkDir`. Leaf setup
seeds `serverHome` with remote fixtures and optionally pre-seeds `agentWorkDir`
to model resume states.

**Participants**

- **remote-agent subprocess** — built from `./cmd/remote-agent`; parses `download`
  and dispatches file or directory downloads through the HTTP client.
- **HTTP client** — `GET /api/files/download` per file plus browse/check helpers
  for recursive directory discovery; Range requests for partial resume.
- **ai-critic-server subprocess** — ephemeral port; `HOME=serverHome`; serves
  download and browse APIs against the fake machine home.
- **serverHome** — temp fake server home; leaf setup pre-creates remote source trees.
- **agentWorkDir** — per-leaf cwd for CLI; leaf setup may pre-create partial local mirrors.
- **agentHome** — temp `HOME` for `~/.ai-critic/remote-agent-config.json` only.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/remote-agent-download-dir-doctest-<id>/` for shared binaries (file lock).

**Behaviors**

- Single-file download streams per-file progress lines without `[N/M]` or `overall`
  suffixes; gains retry and resume like directory mode.
- Directory download walks remote tree via recursive `BrowseDir`, downloads regular
  files (including dotfiles), creates empty subdirectories locally, and resumes into
  existing partial `localDir`.
- Trailing-slash local paths append `basename(remoteDir)` then mirror contents
  under that directory root.
- Relative remote paths join onto server home (`GetHome`).
- CLI stdout lines end with `\n`; directory downloads stream hierarchical progress
  (`[N/M]` item headers, indented `downloaded`/`skipped`/`resumed` sub-lines,
  ` — X% overall` rollup) between banner and summary; empty subdirs emit
  `created <rel>/` lines counted in `[N/M]`.
- Summary includes `N skipped, M resumed` when non-zero.

## Version

0.0.2

## Decision Tree

```
[remote-agent download REMOTE_PATH]
 |
 +-- file-regression/                    (GROUP) single-file path unchanged shape
 |    +-- single-file/                   (LEAF)  hello.txt -> local file
 |
 +-- dir-success/                        (GROUP) directory accepted + mirrored
 |    +-- mirror-tree/                   (LEAF)  remote tree mirrored locally
 |    +-- streams-progress/              (LEAF)  multi-file streaming stdout
 |    +-- resume-skips-complete/          (LEAF)  pre-seeded complete files skipped
 |    +-- resume-partial-file/           (LEAF)  pre-seeded half file resumed
 |    +-- nested-and-dotfiles/           (LEAF)  dotfiles + empty subdir + created line
 |
 +-- dir-rejected/                       (GROUP) remote source invalid
      +-- remote-is-missing/             (LEAF)  remote path absent
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `file-regression/single-file` | Remote file downloads; streaming lines; no `[N/M]`/`overall` |
| 2 | `dir-success/mirror-tree` | Remote tree mirrored into localDir |
| 3 | `dir-success/streams-progress` | Multi-file dir streams `[N/M]`, `overall`, progress before complete |
| 4 | `dir-success/resume-skips-complete` | Pre-seeded complete files → `skipped` in stdout |
| 5 | `dir-success/resume-partial-file` | Pre-seeded half file → `resumed at` + full bytes |
| 6 | `dir-success/nested-and-dotfiles` | Dotfiles, empty `emptydir/`, indexed `created` stdout |
| 7 | `dir-rejected/remote-is-missing` | Non-zero exit, clear error |

## Parameter Coverage

| Factor (significance →) | Leaves |
|-------------------------|--------|
| Source kind (file vs directory) | file-regression/*, dir-success/*, dir-rejected/* |
| Outcome (success vs missing remote) | dir-success/*, dir-rejected/* |
| Resume state (fresh vs skip vs partial) | mirror-tree, resume-skips-complete, resume-partial-file |
| Tree shape (dotfiles, empty dirs, nesting) | nested-and-dotfiles, mirror-tree |
| Streaming progress (hierarchical stdout) | streams-progress, nested-and-dotfiles |
| Single-file regression (no dir markers) | file-regression/single-file |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-download-dir
doctest test ./tests/remote-agent-download-dir/...
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
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

type Request struct {
	Args   []string
	Server string
	Token  string

	// RemotePath is the CLI remote source argument.
	RemotePath string
	// LocalPath is the CLI local destination argument (may be relative to agentWorkDir).
	LocalPath string
	// LocalDir is the resolved local directory root for directory assertions.
	LocalDir string

	// ServerPreseedFiles maps serverHome-relative paths to file contents.
	ServerPreseedFiles map[string]string
	// ServerPreseedDirs lists empty directories to create under serverHome.
	ServerPreseedDirs []string
	// LocalPreseedFiles maps localDir-relative paths to file contents (resume leaves).
	LocalPreseedFiles map[string]string
	// LocalPreseedDirs lists empty directories to create under localDir before download.
	LocalPreseedDirs []string
}

type Response struct {
	ExitCode     int
	Stdout       string
	Stderr       string
	Combined     string
	ServerPort   int
	ServerHome   string
	AgentHome    string
	AgentWorkDir string
	RemotePath   string
	LocalPath    string
	LocalDir     string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		return nil, fmt.Errorf("Request.Args is required (e.g. download <remote> [local])")
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot := findModuleRoot()
	cacheDir := sessionCacheDir()
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	serverHome, err := os.MkdirTemp("", "download-dir-server-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(serverHome) })
	resp.ServerHome = serverHome

	if err := applyServerPreseed(t, serverHome, req.ServerPreseedFiles, req.ServerPreseedDirs); err != nil {
		return nil, err
	}

	agentHome, err := os.MkdirTemp("", "download-dir-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	agentWorkDir, err := os.MkdirTemp("", "download-dir-agent-work-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentWorkDir) })
	resp.AgentWorkDir = agentWorkDir

	if req.LocalDir != "" {
		localDir := req.LocalDir
		if !filepath.IsAbs(localDir) {
			localDir = filepath.Join(agentWorkDir, filepath.FromSlash(localDir))
		}
		req.LocalDir = localDir
		resp.LocalDir = localDir
		if err := applyLocalPreseed(t, localDir, req.LocalPreseedFiles, req.LocalPreseedDirs); err != nil {
			return nil, err
		}
	} else if len(req.LocalPreseedFiles) > 0 || len(req.LocalPreseedDirs) > 0 {
		return nil, fmt.Errorf("LocalDir is required when LocalPreseed* is set")
	}

	credDir := filepath.Join(serverHome, ".ai-critic")
	if err := os.MkdirAll(credDir, 0755); err != nil {
		return nil, err
	}
	credFile := filepath.Join(credDir, "server-credentials")
	if err := os.WriteFile(credFile, []byte(req.Token+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("write credentials: %w", err)
	}

	remoteConfigPath := filepath.Join(agentHome, ".ai-critic", "remote-agent-config.json")
	if err := os.MkdirAll(filepath.Dir(remoteConfigPath), 0755); err != nil {
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

	if err := writeRemoteAgentConfig(remoteConfigPath, normalizedServer, req.Token); err != nil {
		return nil, err
	}

	killPort(serverPort)

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = serverHome
	serverCmd.Env = stripEnvPrefix(os.Environ(), "HOME=")
	serverCmd.Env = stripEnvPrefix(serverCmd.Env, lib.EnvAI_CRITIC_HOME+"=")
	serverCmd.Env = append(serverCmd.Env, "HOME="+serverHome, "AI_CRITIC_NO_OPEN_BROWSER=1")
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
	if err := verifyServerHome(t, normalizedServer, req.Token, serverHome); err != nil {
		return nil, err
	}

	resp.RemotePath = req.RemotePath
	resp.LocalPath = req.LocalPath

	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)
	t.Logf("remote-agent argv: %v (cwd=%s)", argv, agentWorkDir)

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	agentCmd := exec.Command(agentBin, argv...)
	agentCmd.Dir = agentWorkDir
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

	return resp, nil
}

type remoteAgentConfigFile struct {
	Default string            `json:"default,omitempty"`
	Domains []domainConfigRow `json:"domains"`
}

type domainConfigRow struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

func writeRemoteAgentConfig(path, server, token string) error {
	cfg := remoteAgentConfigFile{
		Default: server,
		Domains: []domainConfigRow{{Server: server, Token: token}},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
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
	return 29000 + (hash % 1000)
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

func killPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return
	}
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(out))) {
		_ = exec.Command("kill", "-9", pidStr).Run()
	}
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

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
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

func normalizeAbsPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return eval, nil
}

func verifyServerHome(t *testing.T, serverURL, token, wantHome string) error {
	want, err := normalizeAbsPath(wantHome)
	if err != nil {
		return fmt.Errorf("resolve harness serverHome: %w", err)
	}
	homeURL := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/api/files/home"
	req, err := http.NewRequest(http.MethodGet, homeURL, nil)
	if err != nil {
		return fmt.Errorf("build verify-home request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify server HOME: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read verify-home response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify server HOME status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var home struct {
		Home string `json:"home"`
	}
	if err := json.Unmarshal(data, &home); err != nil {
		return fmt.Errorf("decode home response: %w", err)
	}
	got, err := normalizeAbsPath(home.Home)
	if err != nil {
		return fmt.Errorf("resolve server-reported HOME %q: %w", home.Home, err)
	}
	if got != want {
		return fmt.Errorf(
			"server HOME mismatch: server reports %q (normalized %q) but harness serverHome is %q (normalized %q)",
			home.Home, got, wantHome, want,
		)
	}
	t.Logf("verified server HOME=%s", got)
	return nil
}
```