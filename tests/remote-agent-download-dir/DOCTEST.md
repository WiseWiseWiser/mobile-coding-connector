# Remote-Agent Download Directory Doctests

Doctests for `remote-agent download <REMOTE_PATH> [LOCAL_PATH]` when the remote
source is a file or directory. Directory downloads mirror a remote tree locally
via client-orchestrated per-file GET downloads with streaming progress, transient
retry, and filesystem-based resume (skip complete files, Range for partial files).

Most leaves are **L2 in-process** (`fileupload.RegisterAPIForHome` +
`agentcli.Run`). Two sparse **L3 e2e** smokes keep the product binary path.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `fileupload.RegisterAPIForHome` on a local mux
+ `agentcli.Run` (no product binaries). Two sparse **L3 e2e** smokes keep the
binary path (`UseCLI` + `label: heavy, e2e`): dir success stream + missing remote
reject.

**L2** serves download/browse/home APIs in-process via
`RegisterAPIForHome(mux, serverHome)`. L2 emulates CLI cwd under the agentcli mutex via a temporary
`os.Chdir(agentWorkDir)` (restored after the call; not used for HOME isolation).
**L3 smokes** still run product binaries with `cmd.Dir = agentWorkDir`.

**Participants**

- **L2: agentcli.Run** — in-process `download` CLI against the local mux.
- **L2: fileupload HTTP** — `RegisterAPIForHome` on ephemeral port.
- **L3: remote-agent + ai-critic-server subprocesses** — sparse `UseCLI` smokes
  (`dir-success/streams-progress`, `dir-rejected/remote-is-missing`).
- **serverHome** — temp fake server home; leaf setup pre-creates remote source trees.
- **agentWorkDir** — per-leaf local work root for download destinations and resume preseeds.
- **agentHome** — temp dir for (L3) agent config.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/remote-agent-download-dir-doctest-<id>/` for L3 shared binaries (file lock).

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
- `--dry-run` streams the same hierarchical progress with `would` prefixes and
  `dry-run: download plan` / `dry-run: download complete` banners but performs no
  GET download or local writes; still browses server and stats local files for
  resume preview (`would skip`, `would resume`).

## Version

0.0.3

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
 |    +-- dry-run-mirror/                (LEAF)  --dry-run full plan; no local files
 |    +-- dry-run-resume-preview/        (LEAF)  --dry-run would skip/resume; local unchanged
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
| 8 | `dir-success/dry-run-mirror` | `--dry-run` streams plan; no local files created |
| 9 | `dir-success/dry-run-resume-preview` | Pre-seed partial local; `would skip`/`would resume`; local unchanged |

## Parameter Coverage

| Factor (significance →) | Leaves |
|-------------------------|--------|
| Source kind (file vs directory) | file-regression/*, dir-success/*, dir-rejected/* |
| Outcome (success vs missing remote) | dir-success/*, dir-rejected/* |
| Resume state (fresh vs skip vs partial) | mirror-tree, resume-skips-complete, resume-partial-file |
| Tree shape (dotfiles, empty dirs, nesting) | nested-and-dotfiles, mirror-tree |
| Streaming progress (hierarchical stdout) | streams-progress, nested-and-dotfiles |
| Single-file regression (no dir markers) | file-regression/single-file |
| `--dry-run` (plan only, no mutations) | dry-run-mirror, dry-run-resume-preview |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-download-dir
doctest test ./tests/remote-agent-download-dir/...          # unlabeled L2 mass
doctest test --label e2e ./tests/remote-agent-download-dir/...  # ~2 L3 smokes
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
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/fileupload"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run.
var agentcliInProcessMu sync.Mutex

type Request struct {
	Args   []string
	Server string
	Token  string

	// UseCLI forces the L3 product-binary path. Default false → L2 in-process.
	UseCLI bool
	E2E    bool

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

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseCLI || req.E2E)
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

	LocalFilesBeforeCLI map[string]string
	LocalFilesAfterCLI  map[string]string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		return nil, fmt.Errorf("Request.Args is required (e.g. download <remote> [local])")
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)

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

	resp.RemotePath = req.RemotePath
	resp.LocalPath = req.LocalPath

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, moduleRoot, cacheDir, serverHome, agentHome, agentWorkDir)
	}
	return runInProcessL2(t, d, req, resp, serverHome, agentWorkDir)
}

func runInProcessL2(t *testing.T, d *session.Doctest, req *Request, resp *Response, serverHome, agentWorkDir string) (*Response, error) {
	t.Helper()

	mux := http.NewServeMux()
	fileupload.RegisterAPIForHome(mux, serverHome)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen in-process download server: %w", err)
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
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 10*time.Second); err != nil {
		return nil, err
	}
	if err := verifyServerHome(t, normalizedServer, req.Token, serverHome); err != nil {
		return nil, err
	}
	t.Logf("L2 in-process download server on %s home=%s", normalizedServer, serverHome)

	return finishDownloadRun(t, req, resp, serverURL, agentWorkDir, func(argv []string) (int, string, string, error) {
		// workDir is held under agentcliInProcessMu (same as stdout swap) so
		// relative local destinations resolve without product-binary cmd.Dir.
		return runAgentInProcess(argv, agentWorkDir)
	}, true)
}

func runBinaryE2E(t *testing.T, d *session.Doctest, req *Request, resp *Response, moduleRoot, cacheDir, serverHome, agentHome, agentWorkDir string) (*Response, error) {
	t.Helper()

	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

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

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	return finishDownloadRun(t, req, resp, serverURL, agentWorkDir, func(argv []string) (int, string, string, error) {
		return runAgent(agentBin, argv, agentEnv, agentWorkDir)
	}, false)
}

func finishDownloadRun(
	t *testing.T,
	req *Request,
	resp *Response,
	serverURL, agentWorkDir string,
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
	t.Logf("%s remote-agent argv: %v (work=%s)", mode, argv, agentWorkDir)

	if argsHasDryRun(req.Args) && req.LocalDir != "" {
		resp.LocalFilesBeforeCLI = snapshotDataTree(t, req.LocalDir)
	}

	exitCode, stdout, stderr, runErr := runCLI(argv)
	if runErr != nil {
		return nil, runErr
	}

	if argsHasDryRun(req.Args) && req.LocalDir != "" {
		resp.LocalFilesAfterCLI = snapshotDataTree(t, req.LocalDir)
	}

	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	return resp, nil
}

func runAgentInProcess(argv []string, workDir string) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	if workDir != "" {
		oldWD, err := os.Getwd()
		if err != nil {
			return 0, "", "", err
		}
		if err := os.Chdir(workDir); err != nil {
			return 0, "", "", fmt.Errorf("chdir agentWorkDir: %w", err)
		}
		defer func() { _ = os.Chdir(oldWD) }()
	}

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

func runAgent(bin string, argv, env []string, dir string) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	cmd.Env = env
	cmd.Dir = dir
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
