# Remote-Agent Paste-Bin Doctests

Doctests for `remote-agent paste-bin`: read/write the File Transfer Quick
Transfer scratch pad via `GET/PUT /api/file-transfer/scratch`.

Most leaves are **L2 in-process** (`filetransfer.RegisterAPIForDir` +
`agentcli.Run` with stdin swap). Two sparse **L3 e2e** smokes keep the product
binary path.

# DSN (Domain Specific Notion)

Most leaves are **L2 in-process**: `filetransfer.RegisterAPIForDir` on a local mux
+ `agentcli.Run` with mutex-scoped stdin swap (no product binaries). Two sparse
**L3 e2e** smokes keep the binary path (`UseCLI` + `label: heavy, e2e`):
`write/small-echo` and `rejected/auth-failure`.

**L2** serves scratch APIs in-process via `RegisterAPIForDir(mux, configHome/file-transfer)`
plus a local bearer middleware (valid token = `lib.TestPassword`). **L3 smokes**
still run product binaries with `AI_CRITIC_HOME=configHome`.

**Participants**

- **L2: agentcli.Run** — in-process `paste-bin` CLI; stdin is either a pipe
  (`PipedStdin`) or `/dev/null` (TTY/read mode; char device → not piped).
- **L2: filetransfer HTTP** — `RegisterAPIForDir` on ephemeral port.
- **L3: remote-agent + ai-critic-server subprocesses** — sparse `UseCLI` smokes.
- **configHome** — temp isolated server config; leaf setup seeds or deletes scratch.json.
- **agentHome** — temp dir for (L3) agent config.
- **stdin pipe** — write leaves attach piped stdin; read leaves use DevNull as TTY stand-in.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/remote-agent-paste-bin-doctest-<id>/` for L3 shared binaries (file lock).

**Behaviors**

- TTY stdin (default): **read** — GET scratch, print `content` to stdout; empty scratch
  → silent stdout, exit 0.
- Piped stdin (default): **write** — read raw stdin bytes, PUT scratch (overwrite);
  empty pipe clears scratch (`content: ""`).
- `--read` with piped stdin: force read; stdin bytes ignored for write.
- Write success stderr: green `saved N bytes`, gray preview block (max 3 lines / 200 bytes);
  truncation hint when preview shorter than payload.
- Write stdout echo: full content when `1 ≤ N ≤ 4096`; silent when `N=0` or `N>4096`.
- Invalid UTF-8 stdin: stored as `paste-bin:b64:<base64>` envelope; read decodes to raw bytes.
- `--json`: JSON on stdout only (read or write); no ANSI, no preview/echo.
- `--meta`: gray `updated at` (read) or `saved at` (write) timestamp on stderr.
- `-q` / `--quiet`: suppress stdout echo on small writes.
- Bad token or extra positional args: non-zero exit with actionable errors.

## Version

0.0.3

## Decision Tree

```
[remote-agent paste-bin]
 |
 +-- read/                              (GROUP) TTY read mode
 |    +-- empty/                        (LEAF)  no scratch.json → silent stdout
 |    +-- seeded-utf8/                  (LEAF)  multiline UTF-8 bytes on stdout
 |    +-- json-flag/                    (LEAF)  --json → JSON stdout
 |    +-- meta-flag/                    (LEAF)  --meta → timestamp on stderr
 |
 +-- write/                             (GROUP) piped stdin write mode
 |    +-- small-echo/                   (LEAF)  ≤4096 bytes → stderr saved + stdout echo
 |    +-- empty-pipe/                   (LEAF)  empty stdin clears scratch
 |    +-- large-no-echo/                (LEAF)  >4096 bytes → preview only, no stdout
 |    +-- binary-envelope/              (LEAF)  NUL bytes → b64 envelope round-trip
 |    +-- json-flag/                    (LEAF)  --json → PUT JSON on stdout only
 |    +-- quiet-flag/                   (LEAF)  -q suppresses stdout echo
 |
 +-- mode-override/                     (GROUP) flag precedence over stdin detection
 |    +-- read-force-piped/             (LEAF)  piped stdin + --read → read, API unchanged
 |
 +-- rejected/                          (GROUP) CLI validation and auth errors
      +-- extra-args/                  (LEAF)  positional args rejected
      +-- auth-failure/                 (LEAF)  bad token → non-zero exit
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `read/empty` | Missing scratch.json; exit 0; silent stdout |
| 2 | `read/seeded-utf8` | Seeded multiline UTF-8; stdout matches bytes |
| 3 | `read/json-flag` | `--json` prints scratch JSON on stdout |
| 4 | `read/meta-flag` | `--meta` prints gray `updated at` on stderr |
| 5 | `write/small-echo` | Pipe `hi`; stderr `saved 2 bytes`; stdout echoes `hi` |
| 6 | `write/empty-pipe` | Empty pipe clears scratch; `saved 0 bytes` |
| 7 | `write/large-no-echo` | 5000-byte pipe; stderr preview + truncation; silent stdout |
| 8 | `write/binary-envelope` | NUL stdin; API b64 envelope; decoded bytes match |
| 9 | `write/json-flag` | `--json` on write; JSON only on stdout |
| 10 | `write/quiet-flag` | `-q` suppresses stdout echo for small write |
| 11 | `mode-override/read-force-piped` | Piped junk + `--read`; seeded stdout; API unchanged |
| 12 | `rejected/extra-args` | `paste-bin foo` → non-zero; usage/args hint |
| 13 | `rejected/auth-failure` | Bad `--token` → non-zero exit |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Operation mode (read vs write) | read/*, write/* |
| Stdin source (TTY vs pipe vs empty pipe) | read/*, write/*, mode-override/* |
| `--read` override | mode-override/read-force-piped |
| Output flags (`--json`, `--meta`, `-q`) | read/json-flag, read/meta-flag, write/json-flag, write/quiet-flag |
| Payload size (0, small ≤4096, large >4096) | read/empty, write/small-echo, write/empty-pipe, write/large-no-echo |
| Binary / UTF-8 encoding | read/seeded-utf8, write/binary-envelope |
| Scratch seed state (absent, seeded, stale) | read/empty, read/*, write/empty-pipe, mode-override/* |
| Auth / CLI surface errors | rejected/* |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-paste-bin
doctest test ./tests/remote-agent-paste-bin/...          # unlabeled L2 mass
doctest test --label e2e ./tests/remote-agent-paste-bin/...  # ~2 L3 smokes
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
	"github.com/xhd2015/ai-critic/server/filetransfer"
	"github.com/xhd2015/doctest/session"
)

// agentcliInProcessMu serializes in-process agentcli.Run (stdout/stderr/stdin swaps).
var agentcliInProcessMu sync.Mutex

// ScratchSeed configures scratch.json before the CLI runs.
type ScratchSeed struct {
	Content   string
	UpdatedAt string
}

type Request struct {
	Args   []string
	Server string
	Token  string

	// UseCLI forces the L3 product-binary path. Default false → L2 in-process.
	UseCLI bool
	E2E    bool

	ScratchReset bool
	ScratchSeed  *ScratchSeed

	// PipedStdin nil = TTY (stdin is DevNull / not attached). Non-nil = pipe stdin using StdinBytes
	// (may be empty slice for empty-pipe clears).
	PipedStdin *bool
	StdinBytes []byte
}

func useBinaryPath(req *Request) bool {
	return req != nil && (req.UseCLI || req.E2E)
}

type ScratchEntry struct {
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ServerURL  string
	ConfigHome string
	AgentHome  string
	Token      string

	ScratchAfter ScratchEntry
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if len(req.Args) == 0 {
		req.Args = []string{"paste-bin"}
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	resp.Token = req.Token

	moduleRoot := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
	cacheDir := sessionCacheDir(d.DOCTEST_SESSION_ID)

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	resp.ConfigHome = configHome

	if err := prepareScratch(req, configHome); err != nil {
		return nil, fmt.Errorf("prepare scratch: %w", err)
	}

	agentHome, err := os.MkdirTemp("", "remote-agent-paste-bin-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	if useBinaryPath(req) {
		return runBinaryE2E(t, d, req, resp, moduleRoot, cacheDir, configHome, agentHome)
	}
	return runInProcessL2(t, d, req, resp, configHome)
}

func transferDir(configHome string) string {
	return filepath.Join(configHome, "file-transfer")
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

func runInProcessL2(t *testing.T, d *session.Doctest, req *Request, resp *Response, configHome string) (*Response, error) {
	t.Helper()

	mux := http.NewServeMux()
	filetransfer.RegisterAPIForDir(mux, transferDir(configHome))
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Auth against fixed test password so auth-failure leaves (bad req.Token) still work
	// and post-CLI scratch fetch can use lib.TestPassword.
	handler := withBearerAuth(lib.TestPassword, mux)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen in-process paste-bin server: %w", err)
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
	resp.ServerURL = normalizedServer

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 10*time.Second); err != nil {
		return nil, err
	}
	t.Logf("L2 in-process paste-bin server on %s configHome=%s", normalizedServer, configHome)

	return finishPasteRun(t, req, resp, serverURL, func(argv []string) (int, string, string, error) {
		return runAgentInProcess(argv, req.PipedStdin, req.StdinBytes)
	}, true)
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
	resp.ServerURL = normalizedServer

	configPath := filepath.Join(agentHome, ".ai-critic", "remote-agent-config.json")
	if err := writeRemoteAgentConfig(configPath, normalizedServer, req.Token); err != nil {
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

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	return finishPasteRun(t, req, resp, serverURL, func(argv []string) (int, string, string, error) {
		return runAgent(agentBin, argv, agentEnv, req.PipedStdin, req.StdinBytes)
	}, false)
}

func finishPasteRun(
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

	// Post-CLI verification always uses server credentials so auth-failure leaves
	// can still assert scratch state was not mutated.
	entry, err := fetchScratchEntry(strings.TrimRight(serverURL, "/"), lib.TestPassword)
	if err != nil {
		return nil, fmt.Errorf("fetch scratch after CLI: %w", err)
	}
	resp.ScratchAfter = entry
	return resp, nil
}

func runAgentInProcess(argv []string, pipedStdin *bool, stdinBytes []byte) (int, string, string, error) {
	agentcliInProcessMu.Lock()
	defer agentcliInProcessMu.Unlock()

	// Stdin: pipe for write mode; /dev/null (char device) for TTY/read mode.
	oldIn := os.Stdin
	var stdinFile *os.File
	if pipedStdin != nil {
		r, w, err := os.Pipe()
		if err != nil {
			return 0, "", "", err
		}
		go func() {
			_, _ = w.Write(stdinBytes)
			_ = w.Close()
		}()
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

func runAgent(bin string, argv, env []string, pipedStdin *bool, stdinBytes []byte) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	cmd.Env = env
	if pipedStdin != nil {
		cmd.Stdin = bytes.NewReader(stdinBytes)
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
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
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
	return 27000 + (hash % 1000)
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

func fetchScratchEntry(serverURL, token string) (ScratchEntry, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(serverURL, "/")+"/api/file-transfer/scratch", nil)
	if err != nil {
		return ScratchEntry{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ScratchEntry{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ScratchEntry{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return ScratchEntry{}, fmt.Errorf("scratch GET status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var entry ScratchEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return ScratchEntry{}, err
	}
	return entry, nil
}
```
