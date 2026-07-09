# Remote-Agent Upload Directory Doctests

End-to-end tests for `remote-agent upload <LOCAL_PATH> [REMOTE_PATH]` when the
local source is a file or directory. Directory uploads mirror a local tree onto
the server via client-orchestrated per-file chunked uploads, with a pre-flight
guard that accepts only missing or completely empty remote destinations.

# DSN (Domain Specific Notion)

The harness exercises the remote-agent CLI against a real `ai-critic-server`
subprocess. The server runs with `HOME=serverHome` so remote paths resolve under
an isolated fake machine home. The CLI runs in a separate `agentHome` with only
`remote-agent-config.json`. Leaf setup seeds `localDir` fixtures on the test host
and optionally pre-seeds `serverHome` to model destination states.

**Participants**

- **remote-agent subprocess** — built from `./cmd/remote-agent`; parses `upload`
  and dispatches file or directory uploads through the HTTP client.
- **HTTP client** — existing chunked upload API (`/api/files/upload/{init,chunk,complete}`)
  plus browse/check helpers for destination guards.
- **ai-critic-server subprocess** — ephemeral port; `HOME=serverHome`; serves
  upload and browse APIs against the fake machine home.
- **localDir / local file** — leaf `Setup` creates temp trees or files on the CLI host.
- **serverHome** — temp fake server home; leaf setup may pre-create destination paths.
- **agentHome** — temp `HOME` for `~/.ai-critic/remote-agent-config.json` only.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/remote-agent-upload-dir-doctest-<id>/` for shared binaries (file lock).

**Behaviors**

- Single-file upload keeps current behavior: one chunked session, success line with
  path and size, executable bit preserved when local mode has any execute bit.
- Directory upload walks regular files (including dotfiles), skips symlinks/devices,
  creates empty subdirectories remotely, and rejects destinations that exist as
  non-empty directories or as regular files before transferring bytes.
- Trailing-slash remote paths append `basename(localDir)` then mirror contents
  under that directory root.
- Relative remote paths join onto server home (`GetHome`).
- CLI stdout lines end with `\n`; directory uploads stream hierarchical progress
  (`[N/M]` item headers, indented chunk sub-lines, ` — X% overall` rollup) between
  banner and summary; empty subdirs emit `created <rel>/` lines counted in `[N/M]`.
- Single-file uploads keep flat chunk lines without `[N/M]` or `overall` suffixes.
- `--dry-run` streams the same hierarchical progress with `would` prefixes and
  `dry-run: upload plan` / `dry-run: upload complete` banners but performs no
  server mkdir, upload init/chunk/complete; destination guard still runs.

## Version

0.0.2

## Decision Tree

```
[remote-agent upload LOCAL_PATH]
 |
 +-- file-regression/                    (GROUP) single-file path unchanged
 |    +-- single-file/                   (LEAF)  hello.txt -> remote file
 |    +-- dry-run-single-file/           (LEAF)  --dry-run single file; no overall
 |
 +-- dir-success/                        (GROUP) directory accepted + mirrored
 |    +-- dst-not-exists/                (LEAF)  missing remoteDir
 |    +-- dst-empty-exists/              (LEAF)  empty remoteDir
 |    +-- nested-and-dotfiles/           (LEAF)  dotfiles + empty subdir + created line
 |    +-- streams-progress/              (LEAF)  multi-file streaming stdout
 |    +-- trailing-slash-dst/            (LEAF)  parent/ -> parent/proj/
 |    +-- dry-run-mirror/                (LEAF)  --dry-run dir plan; serverHome unchanged
 |
 +-- dir-rejected/                       (GROUP) pre-flight guard fails
      +-- dst-has-file/                  (LEAF)  remoteDir contains a file
      +-- dst-has-subdir/                (LEAF)  remoteDir contains a subdir
      +-- dst-is-file/                   (LEAF)  remoteDir is a regular file
      +-- dry-run-guard-fails/           (LEAF)  --dry-run + non-empty dst; server unchanged
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `file-regression/single-file` | Local file `hello.txt` uploads unchanged |
| 2 | `dir-success/dst-not-exists` | Mirror tree when remote destination absent |
| 3 | `dir-success/dst-empty-exists` | Mirror tree into empty existing remoteDir |
| 4 | `dir-success/nested-and-dotfiles` | Dotfiles, empty `emptydir/`, indexed `created` stdout |
| 5 | `dir-success/streams-progress` | Multi-file dir streams `[N/M]`, `overall`, chunk lines |
| 6 | `dir-success/trailing-slash-dst` | `parent/` places files under `parent/proj/` |
| 7 | `dir-rejected/dst-has-file` | Reject when `remoteDir/existing.txt` present |
| 8 | `dir-rejected/dst-has-subdir` | Reject when `remoteDir/child/` present |
| 9 | `dir-rejected/dst-is-file` | Reject when destination path is a file |
| 10 | `dir-success/dry-run-mirror` | `--dry-run` dir plan; `would upload`; serverHome unchanged |
| 11 | `dir-rejected/dry-run-guard-fails` | `--dry-run` with non-empty dst; error; server unchanged |
| 12 | `file-regression/dry-run-single-file` | `--dry-run` single file; `would upload chunk`; no `overall` |

## Parameter Coverage

| Factor (significance →) | Leaves |
|-------------------------|--------|
| Source kind (file vs directory) | file-regression/*, dir-success/*, dir-rejected/* |
| Outcome (success vs guard reject) | dir-success/*, dir-rejected/* |
| Remote destination state | dst-not-exists, dst-empty-exists, dst-has-file, dst-has-subdir, dst-is-file |
| Tree shape (dotfiles, empty dirs, nesting) | nested-and-dotfiles, dst-not-exists, dst-empty-exists |
| Streaming progress (hierarchical stdout) | streams-progress, nested-and-dotfiles |
| Remote path form (plain vs trailing `/`) | trailing-slash-dst |
| `--dry-run` (plan only, no mutations) | dry-run-mirror, dry-run-guard-fails, dry-run-single-file |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-upload-dir
doctest test ./tests/remote-agent-upload-dir/...
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

	// LocalPath is the absolute local file or directory passed to upload.
	LocalPath string
	// RemotePath is the optional CLI remote destination argument.
	RemotePath string
	// RemoteDir is the resolved remote directory root for directory assertions.
	RemoteDir string

	// ServerPreseedFiles maps serverHome-relative paths to file contents.
	ServerPreseedFiles map[string]string
	// ServerPreseedDirs lists empty directories to create under serverHome.
	ServerPreseedDirs []string
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ServerHome string
	AgentHome  string
	LocalPath  string
	RemotePath string
	RemoteDir  string

	// ServerFilesBeforeCLI / ServerFilesAfterCLI snapshot serverHome file
	// contents (serverHome-relative paths) when Args include --dry-run.
	ServerFilesBeforeCLI map[string]string
	ServerFilesAfterCLI  map[string]string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		return nil, fmt.Errorf("Request.Args is required (e.g. upload <local> [remote])")
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot := findModuleRoot()
	cacheDir := sessionCacheDir()
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	serverHome, err := os.MkdirTemp("", "upload-dir-server-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(serverHome) })
	resp.ServerHome = serverHome

	if err := applyServerPreseed(t, serverHome, req.ServerPreseedFiles, req.ServerPreseedDirs); err != nil {
		return nil, err
	}

	agentHome, err := os.MkdirTemp("", "upload-dir-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

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

	if req.LocalPath != "" {
		absLocal, err := filepath.Abs(req.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("abs local path: %w", err)
		}
		req.LocalPath = absLocal
		resp.LocalPath = absLocal
	}
	resp.RemotePath = req.RemotePath
	resp.RemoteDir = req.RemoteDir

	argv := []string{"--server", serverURL, "--token", req.Token}
	argv = append(argv, req.Args...)
	t.Logf("remote-agent argv: %v", argv)

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	if argsHasDryRun(req.Args) {
		resp.ServerFilesBeforeCLI = snapshotDataTree(t, serverHome)
	}

	agentCmd := exec.Command(agentBin, argv...)
	agentCmd.Env = agentEnv

	var stdout, stderr bytes.Buffer
	agentCmd.Stdout = &stdout
	agentCmd.Stderr = &stderr

	runErr := agentCmd.Run()

	if argsHasDryRun(req.Args) {
		resp.ServerFilesAfterCLI = snapshotDataTree(t, serverHome)
	}
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
	return 28000 + (hash % 1000)
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